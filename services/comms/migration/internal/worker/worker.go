package worker

import (
	"context"
	"fmt"

	"era/services/comms/internal/imapclient"
	"era/services/comms/migration/internal/audit"
	"era/services/comms/migration/internal/importers/imap"
	"era/services/comms/migration/internal/jobs"
	"era/services/comms/migration/internal/target"
)

// Runner executes migration jobs asynchronously.
type Runner struct {
	Jobs  jobs.Repository
	Audit audit.Recorder
}

// JobRequest — network migration parameters.
type JobRequest struct {
	Source     string
	Mailbox    string
	SourceIMAP imap.NetworkConfig
	Target     target.Writer
	Folder     string
	AllFolders bool
	Mode       string
}

func (r *Runner) Start(ctx context.Context, req JobRequest) (jobs.Job, error) {
	j := r.Jobs.CreateQueued(req.Source, req.Mailbox)
	go r.run(ctx, j.ID, req)
	return j, nil
}

func (r *Runner) run(ctx context.Context, jobID string, req JobRequest) {
	r.Jobs.SetStatus(jobID, "running")
	var (
		msgs []imapclient.Message
		err  error
	)
	switch {
	case req.AllFolders || req.Folder == "*":
		msgs, err = imap.ImportNetworkAll(req.SourceIMAP)
	default:
		msgs, err = imap.ImportNetwork(req.SourceIMAP, req.Folder)
	}
	if err != nil {
		r.Jobs.Fail(jobID, err.Error())
		r.Audit.Record(audit.Event{JobID: jobID, Action: "MIGRATION_FAILED", Detail: err.Error()})
		return
	}
	ok, fail := 0, 0
	for _, msg := range msgs {
		uidKey := fmt.Sprintf("%s:%s:%d", req.Mailbox, msg.Folder, msg.UID)
		if req.Mode == "delta" && r.Jobs.Seen(uidKey) {
			continue
		}
		if err := req.Target.Write(msg); err != nil {
			fail++
			r.Audit.Record(audit.Event{JobID: jobID, Action: "MIGRATION_ITEM_FAIL", SourceUID: uidKey, Detail: err.Error()})
			continue
		}
		r.Jobs.MarkSeen(uidKey)
		ok++
		r.Audit.Record(audit.Event{JobID: jobID, Action: "MIGRATION_ITEM_OK", SourceUID: uidKey})
	}
	_ = req.Target.Close()
	r.Jobs.Complete(jobID, len(msgs), ok, fail)
	r.Audit.Record(audit.Event{JobID: jobID, Action: "MIGRATION_JOB_DONE"})
}

// ImportFileJob — legacy file-line import for golden tests.
func (r *Runner) ImportFileJob(source, mailbox, imapFile, archiveFile string) jobs.Job {
	total := 0
	if imapFile != "" {
		items, err := imap.ImportMailbox(imapFile)
		if err == nil {
			total += len(items)
		}
	}
	j := r.Jobs.CreateDone(source, mailbox, total)
	r.Audit.Record(audit.Event{JobID: j.ID, Action: "MIGRATION_JOB_CREATED"})
	return j
}

// MockTarget collects writes for tests.
type MockTarget struct {
	Written []imapclient.Message
}

func (m *MockTarget) Name() string { return "mock" }
func (m *MockTarget) Write(msg imapclient.Message) error {
	m.Written = append(m.Written, msg)
	return nil
}
func (m *MockTarget) Close() error { return nil }
