package reporting

import (
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/git"
	"sync"
)

type Report struct {
	GitBranch string
	GitCommit string
	GitTag string

	Target string

	Success bool
	Jobs []*job.Job

	lock sync.Mutex
}

func New(target string) (Report, error) {
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
