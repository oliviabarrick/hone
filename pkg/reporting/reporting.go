package reporting

import (
	"github.com/justinbarrick/hone/pkg/cache"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/git"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/justinbarrick/hone/pkg/scm"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sync"
)

type Report struct {
	GitBranch string
	GitCommit string
	GitTag string

	Target string

	Success bool
	Jobs []*job.Job

	scms []*scm.SCM
	cache cache.Cache
	lock sync.Mutex
}

func New(target string, scms []*scm.SCM, cache cache.Cache) (Report, error) {
	repo, err := git.NewRepository()
	if err != nil {
		return Report{}, err
	}

	branch, _ := repo.Branch()
	commit, _ := repo.Commit()
	tag, _ := repo.Tag()

	return Report{
		GitBranch: branch,
		GitCommit: commit,
		GitTag: tag,
		Target: target,
		cache: cache,
		scms: scms,
	}, nil
}

func (r *Report) ReportJob(callback func(*job.Job) error) func(*job.Job) error {
	return func(j *job.Job) error {
		r.lock.Lock()
		r.Jobs = append(r.Jobs, j)
		r.lock.Unlock()

		return callback(j)
	}
}

func (r *Report) UploadReport() error {
	if r.cache == nil || ! r.cache.Enabled() {
		return nil
	}

	reportJson, reportJsonUrl, err := r.cache.Writer("report-blobs", "report.json")
	if err != nil {
		return err
	}

	err = json.NewEncoder(reportJson).Encode(r)
	if err != nil {
		return err
	}

	reportJson.Close()

	reportWriter, reportUrl, err := r.cache.Writer("reports", "report.html")
	if err != nil {
		return err
	}

	template.Must(template.ParseFiles("index.html")).Execute(reportWriter, struct{
		ReportJSON string
	}{
		ReportJSON: reportJsonUrl,
	})

	reportWriter.Close()

	logger.Printf("Report uploaded to: %s", reportUrl)
	return nil
}

func (r *Report) Exit(errs ...error) {
	r.Success = len(errs) == 0

	err := r.UploadReport()
	if err != nil {
		logger.Errorf("Error uploading report to cache: %s", err)
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		if errs[0].Error() == fmt.Sprintf("Target %s not found.", r.Target) {
			logger.Printf("Error: Target %s not found in configuration!", r.Target)
		}

		logger.Errorf("Exiting with failure.")
	} else {
		logger.Successf("Build completed successfully!")
	}

	err = scm.ReportBuild(r.scms, r.Success)
	if err != nil {
		logger.Errorf("Error reporting build to SCM: %s", err)
		errs = append(errs, err)
	}

	os.Exit(len(errs))
}
