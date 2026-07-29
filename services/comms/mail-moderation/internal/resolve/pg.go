package resolve

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PGCurators — таблица moderation_curators.
type PGCurators struct {
	db *sql.DB
}

func OpenPGCurators(dsn string) (*PGCurators, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PGCurators{db: db}, nil
}

func (p *PGCurators) Curator(sender string) (string, error) {
	var email string
	err := p.db.QueryRow(`SELECT curator_email FROM moderation_curators WHERE lower(sender_email)=lower($1)`, sender).Scan(&email)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return email, err
}

func (p *PGCurators) Set(sender, curator string) error {
	_, err := p.db.Exec(`INSERT INTO moderation_curators (sender_email, curator_email, updated_at)
		VALUES ($1,$2,now()) ON CONFLICT (sender_email) DO UPDATE SET curator_email=EXCLUDED.curator_email, updated_at=now()`,
		strings.ToLower(sender), curator)
	return err
}

func (p *PGCurators) AddNoviceGroup(_ string) error { return nil }

// OverlayDir — curator из PG, остальное из base Directory.
type OverlayDir struct {
	Base     Directory
	Curators *PGCurators
}

func (o *OverlayDir) Manager(sender string) (string, error) {
	if o.Base != nil {
		return o.Base.Manager(sender)
	}
	return "", nil
}

func (o *OverlayDir) Attr(sender, name string) (string, error) {
	if o.Base != nil {
		return o.Base.Attr(sender, name)
	}
	return "", nil
}

func (o *OverlayDir) Curator(sender string) (string, error) {
	if o.Curators != nil {
		if v, err := o.Curators.Curator(sender); err != nil {
			return "", err
		} else if v != "" {
			return v, nil
		}
	}
	if o.Base != nil {
		return o.Base.Curator(sender)
	}
	return "", nil
}

// OpenCuratorsFromEnv — nil если нет DSN.
func OpenCuratorsFromEnv() (*PGCurators, error) {
	dsn := os.Getenv("ERA_MM_POSTGRES_DSN")
	if dsn == "" {
		return nil, nil
	}
	return OpenPGCurators(dsn)
}
