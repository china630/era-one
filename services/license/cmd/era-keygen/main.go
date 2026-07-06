// Command era-keygen — вендор-сторонний инструмент лицензирования ERA XDR (ADR-0010).
//
// Подкоманды:
//   genkey  — создать пару ключей вендора (приватный хранить в HSM/KMS!)
//   issue   — выпустить лицензию (помодульно, на 1/3 года, с привязкой)
//   verify  — проверить токен публичным ключом и оценить статус
//
// Примеры:
//   era-keygen genkey -out ./keys
//   era-keygen issue -priv ./keys/vendor.key -customer "Bank A" -tenant t1 \
//       -modules vm,ai,response -nodes 50000 -years 3 -deployment deploy-XYZ
//   era-keygen verify -pub ./keys/vendor.pub -token <TOKEN> -deployment deploy-XYZ -nodes 100
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"era/services/license/internal/license"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "genkey":
		err = cmdGenkey(os.Args[2:])
	case "issue":
		err = cmdIssue(os.Args[2:])
	case "issue-lease":
		err = cmdIssueLease(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "revoke":
		err = cmdRevoke(os.Args[2:])
	case "fingerprint":
		err = cmdFingerprint(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "неизвестная подкоманда: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`era-keygen — лицензирование ERA XDR (ADR-0010)

Подкоманды:
  genkey  -out DIR
  issue   -priv KEY -customer NAME -tenant ID [-edition core]
          [-modules vm,ai,response,federated,national]
          [-bundle core-ai-response|full-upsell]  (GA-1: core-ai-response = ai+response)
          [-years 1|3] [-deployment ID] [-grace 30]
  issue-lease -priv KEY -lid LICENSE_ID -deployment ID -tenant ID
          [-modules ai,response] [-days 30] [-grace 30] [-offline-max 90] [-renewal-h 24]
  verify  -pub KEY -token TOKEN [-deployment ID] [-nodes N]
  revoke  -priv KEY -lids lic-1,lic-2 [-out crl.token]
  fingerprint -machine MID -board SERIAL [-disks d1,d2] [-macs m1,m2] [-salt SALT]

KEY может быть путём к файлу или строкой base64.`)
}

// ── genkey ────────────────────────────────────────────────────────────────────

func cmdGenkey(args []string) error {
	out := flagValue(args, "-out", "./keys")
	pub, priv, err := license.GenerateKeypair()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		return err
	}
	privPath := filepath.Join(out, "vendor.key")
	pubPath := filepath.Join(out, "vendor.pub")
	if err := os.WriteFile(privPath, []byte(license.EncodeKey(priv)), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(pubPath, []byte(license.EncodeKey(pub)), 0o644); err != nil {
		return err
	}
	fmt.Printf("Пара ключей создана:\n  приватный: %s  (ХРАНИТЬ В СЕКРЕТЕ / HSM)\n  публичный: %s  (встраивается в control-plane)\n", privPath, pubPath)
	return nil
}

// ── issue ─────────────────────────────────────────────────────────────────────

func cmdIssue(args []string) error {
	privArg := flagValue(args, "-priv", "")
	if privArg == "" {
		return fmt.Errorf("-priv обязателен (путь к файлу или base64)")
	}
	priv, err := loadPrivate(privArg)
	if err != nil {
		return err
	}

	customer := flagValue(args, "-customer", "")
	tenant := flagValue(args, "-tenant", "")
	if customer == "" || tenant == "" {
		return fmt.Errorf("-customer и -tenant обязательны")
	}
	edition := flagValue(args, "-edition", "core")
	years := flagInt(args, "-years", 1)
	if years != 1 && years != 3 {
		return fmt.Errorf("-years должен быть 1 или 3")
	}
	nodes := flagInt(args, "-nodes", 0)
	grace := flagInt(args, "-grace", 30)
	deployment := flagValue(args, "-deployment", "")

	modules, err := resolveModules(flagValue(args, "-bundle", ""), flagValue(args, "-modules", ""))
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	lid, err := newLicenseID()
	if err != nil {
		return err
	}
	claims := &license.Claims{
		LicenseID:  lid,
		Customer:   customer,
		TenantID:   tenant,
		Edition:    edition,
		Modules:    modules,
		MaxNodes:   nodes,
		Deployment: deployment,
		IssuedAt:   now.Unix(),
		NotBefore:  now.Unix(),
		ExpiresAt:  now.AddDate(years, 0, 0).Unix(),
		GraceDays:  grace,
	}

	token, err := license.Sign(claims, priv)
	if err != nil {
		return err
	}

	fmt.Printf("Лицензия выпущена (lid=%s, до %s, узлов=%d):\n\n%s\n",
		lid, now.AddDate(years, 0, 0).Format("2006-01-02"), nodes, token)
	return nil
}

// ── issue-lease (ADR-0018 §6) ─────────────────────────────────────────────────

func cmdIssueLease(args []string) error {
	privArg := flagValue(args, "-priv", "")
	lid := flagValue(args, "-lid", "")
	deployment := flagValue(args, "-deployment", "")
	tenant := flagValue(args, "-tenant", "")
	if privArg == "" || lid == "" || deployment == "" || tenant == "" {
		return fmt.Errorf("-priv, -lid, -deployment и -tenant обязательны")
	}
	priv, err := loadPrivate(privArg)
	if err != nil {
		return err
	}
	modules, err := parseModules(flagValue(args, "-modules", ""))
	if err != nil {
		return err
	}
	days := flagInt(args, "-days", 30)
	grace := flagInt(args, "-grace", 30)
	offlineMax := flagInt(args, "-offline-max", 90)
	renewalH := flagInt(args, "-renewal-h", 24)

	now := time.Now().UTC()
	def := license.DefaultLeasePolicy()
	claims := &license.LeaseClaims{
		LicenseID:            lid,
		DeploymentID:         deployment,
		TenantID:             tenant,
		Modules:              modules,
		IssuedAt:             now.Unix(),
		ExpiresAt:            now.AddDate(0, 0, days).Unix(),
		GraceDays:            grace,
		OfflineMaxDays:       offlineMax,
		RenewalIntervalHours: renewalH,
		DegradationMode:      def.DegradationMode,
	}
	if grace == 0 {
		claims.GraceDays = def.GraceDays
	}

	token, err := license.SignLease(claims, priv)
	if err != nil {
		return err
	}
	fmt.Printf("Lease выпущен (lid=%s, deployment=%s, до %s):\n\n%s\n",
		lid, deployment, now.AddDate(0, 0, days).Format("2006-01-02"), token)
	return nil
}

// ── verify ────────────────────────────────────────────────────────────────────

func cmdVerify(args []string) error {
	pubArg := flagValue(args, "-pub", "")
	token := flagValue(args, "-token", "")
	if pubArg == "" || token == "" {
		return fmt.Errorf("-pub и -token обязательны")
	}
	pub, err := loadPublic(pubArg)
	if err != nil {
		return err
	}
	claims, err := license.Verify(token, pub)
	if err != nil {
		return err
	}
	deployment := flagValue(args, "-deployment", "")
	nodes := flagInt(args, "-nodes", 0)
	ev := claims.Evaluate(time.Now().UTC(), deployment, nodes)

	out, _ := json.MarshalIndent(struct {
		Claims     *license.Claims      `json:"claims"`
		Evaluation license.Evaluation   `json:"evaluation"`
	}{claims, ev}, "", "  ")
	fmt.Println(string(out))
	if ev.Status != license.StatusValid && ev.Status != license.StatusGrace {
		os.Exit(3)
	}
	return nil
}

// ── revoke ────────────────────────────────────────────────────────────────────

func cmdRevoke(args []string) error {
	privArg := flagValue(args, "-priv", "")
	lids := flagValue(args, "-lids", "")
	if privArg == "" || strings.TrimSpace(lids) == "" {
		return fmt.Errorf("-priv и -lids обязательны")
	}
	priv, err := loadPrivate(privArg)
	if err != nil {
		return err
	}
	var revoked []string
	for _, s := range strings.Split(lids, ",") {
		if s = strings.TrimSpace(s); s != "" {
			revoked = append(revoked, s)
		}
	}
	crl := &license.CRL{IssuedAt: time.Now().UTC().Unix(), Revoked: revoked}
	token, err := license.SignCRL(crl, priv)
	if err != nil {
		return err
	}
	if out := flagValue(args, "-out", ""); out != "" {
		if err := os.WriteFile(out, []byte(token), 0o644); err != nil {
			return err
		}
		fmt.Printf("CRL (%d записей) сохранён в %s\n", len(revoked), out)
		return nil
	}
	fmt.Printf("CRL (%d записей):\n\n%s\n", len(revoked), token)
	return nil
}

// ── fingerprint (demo/manual) ───────────────────────────────────────────────────

func cmdFingerprint(args []string) error {
	sig := license.Signals{
		MachineID:   flagValue(args, "-machine", ""),
		BoardSerial: flagValue(args, "-board", ""),
		DiskSerials: splitCSV(flagValue(args, "-disks", "")),
		MACs:        splitCSV(flagValue(args, "-macs", "")),
	}
	salt := flagValue(args, "-salt", "era-xdr")
	fmt.Println(license.ComposeFingerprint(sig, salt))
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ── helpers ────────────────────────────────────────────────────────────────────

func resolveModules(bundle, csv string) ([]license.Module, error) {
	if bundle != "" {
		if csv != "" {
			return nil, fmt.Errorf("укажите либо -bundle, либо -modules, не оба")
		}
		return license.ModulesForBundle(license.Bundle(bundle))
	}
	return parseModules(csv)
}

func parseModules(csv string) ([]license.Module, error) {
	if strings.TrimSpace(csv) == "" {
		return nil, nil
	}
	known := map[license.Module]bool{}
	for _, m := range license.KnownModules {
		known[m] = true
	}
	var out []license.Module
	for _, part := range strings.Split(csv, ",") {
		m := license.Module(strings.TrimSpace(part))
		if !known[m] {
			return nil, fmt.Errorf("неизвестный модуль: %q (доступны: vm,ai,response,federated,national)", part)
		}
		out = append(out, m)
	}
	return out, nil
}

func newLicenseID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "lic-" + hex.EncodeToString(b), nil
}

func loadPrivate(arg string) ([]byte, error) {
	s, err := readKeyArg(arg)
	if err != nil {
		return nil, err
	}
	k, err := license.DecodePrivateKey(s)
	return k, err
}

func loadPublic(arg string) ([]byte, error) {
	s, err := readKeyArg(arg)
	if err != nil {
		return nil, err
	}
	k, err := license.DecodePublicKey(s)
	return k, err
}

// readKeyArg возвращает содержимое файла, если arg — путь, иначе сам arg (base64).
func readKeyArg(arg string) (string, error) {
	if info, err := os.Stat(arg); err == nil && !info.IsDir() {
		b, err := os.ReadFile(arg)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return strings.TrimSpace(arg), nil
}

// flagValue — простой парсер "-flag value" без зависимости от порядка.
func flagValue(args []string, name, def string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return def
}

func flagInt(args []string, name string, def int) int {
	s := flagValue(args, name, "")
	if s == "" {
		return def
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return def
	}
	return v
}
