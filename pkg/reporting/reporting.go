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
	"path/filepath"
	"sync"
	"time"
)

//go:generate go-bindata -pkg reporting -nomemcopy templates/...

type Report struct {
	GitBranch string
	GitCommit string
	GitTag string

	Target string

	StartTime time.Time
	EndTime time.Time

	Success bool
	Jobs []*job.Job

	LogURL string

	scms []*scm.SCM
	cache cache.Cache
	lock sync.Mutex
}

func New(target string, scms []*scm.SCM, cache cache.Cache) (Report, error) {
	repo, _ := git.NewRepository()

	branch, _ := repo.Branch()
	commit, _ := repo.Commit()
	tag, _ := repo.Tag()

	return Report{
		GitBranch: branch,
		GitCommit: commit,
		GitTag: tag,
		Target: target,
		StartTime: time.Now().UTC(),
		cache: cache,
		scms: scms,
	}, nil
}

func (r *Report) SetLogURL(url string) {
	r.LogURL = url
}

func (r *Report) ReportJob(callback func(*job.Job) error) func(*job.Job) error {
	return func(j *job.Job) error {
		r.lock.Lock()
		r.Jobs = append(r.Jobs, j)
		r.lock.Unlock()

		return callback(j)
	}
}

func (r *Report) SetCache(cache cache.Cache) {
	r.cache = cache
}

func (r *Report) UploadReport() (string, error) {
	if r.cache == nil || ! r.cache.Enabled() {
		return "", nil
	}

	r.EndTime = time.Now().UTC()

	base := filepath.Join(r.GitCommit, fmt.Sprintf("%d", r.StartTime.Unix()))

	reportJson, reportJsonUrl, err := r.cache.Writer("report-blobs", filepath.Join(base, "report.json"))
	if err != nil {
		return "", err
	}

	err = json.NewEncoder(reportJson).Encode(r)
	if err != nil {
		return "", err
	}

	reportJson.Close()

	reportWriter, reportUrl, err := r.cache.Writer("reports", filepath.Join(base, "report.html"))
	if err != nil {
		return "", err
	}

	data, err := Asset("templates/index.html")
	if err != nil {
		return "", err
	}

	template.Must(template.New("").Parse(string(data))).Execute(reportWriter, struct{
		ReportJSON string
		LogURL string
	}{
		ReportJSON: reportJsonUrl,
		LogURL: r.LogURL,
	})

	reportWriter.Close()

	logger.Printf("Report uploaded to: %s", reportUrl)
	return reportUrl, nil
}

func (r *Report) Final(errs ...error) {
	r.Success = len(errs) == 0

	reportUrl, err := r.UploadReport()
	if err != nil {
		logger.Errorf("Error uploading report to cache: %s", err)
		errs = append(errs, err)
	}

	if r.LogURL != "" {
		logger.Printf("Logs available: %s", r.LogURL)
	}

	if len(errs) > 0 {
		if errs[0].Error() == fmt.Sprintf("Target %s not found.", r.Target) {
			logger.Printf("Error: Target %s not found in configuration!", r.Target)
		}

		logger.Errorf("Exiting with failure.")
	} else {
		logger.Successf("Build completed successfully!")
	}

	err = scm.ReportBuild(r.scms, r.Success, reportUrl)
	if err != nil {
		logger.Errorf("Error reporting build to SCM: %s", err)
		errs = append(errs, err)
	}
}

func (r *Report) Exit(errs ...error) {
	r.Final(errs...)
	os.Exit(len(errs))
}
