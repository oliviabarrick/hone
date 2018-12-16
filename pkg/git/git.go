package git

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
//	"github.com/davecgh/go-spew/spew"
)

func CleanRepoUrl(repoUrl string) string {
	notAllowed := regexp.MustCompile("[^a-z:/]+")
	unwanted := regexp.MustCompile("(^[^@]+@|[.]git$)")

	url := repoUrl
	url = unwanted.ReplaceAllString(url, "")
	url = notAllowed.ReplaceAllString(url, "_")
	url = strings.Replace(url, ":", "_", -1)
	url = strings.Replace(url, "/", "_", -1)

	return url
}

type Repository struct {
	Repo *git.Repository
}

func NewRepository() (Repository, error) {
	var r *git.Repository

	dir, err := os.Getwd()
	if err != nil {
		return Repository{}, err
	}

	for r == nil {
		r, err = git.PlainOpen(dir)
		if err != nil {
			if dir == "/" {
				return Repository{}, err
			}
		} else {
			break
		}

		if dir == "/" {
			break
		}

		dir = filepath.Dir(dir)
	}

	return Repository{
		Repo: r,
	}, nil
}

func (r Repository) Tag() (string, error) {
	commit, err := r.Commit()
	if err != nil {
		return "", err
	}

	tags, err := r.Repo.Tags()
	if err != nil {
		return "", err
	}

	tag := ""
	err = tags.ForEach(func (ref *plumbing.Reference) error {
		if ref.Hash().String() != commit {
			return nil
		}

		tag = ref.Name().Short()
		return nil
	})
	if err != nil {
		return "", err
	}

	return tag, nil
}

func (r Repository) Commit() (string, error) {
	if r.Repo == nil {
		return "", errors.New("Repository not set.")
	}

	head, err := r.Repo.Head()
	if err != nil {
		return "", err
	}

	commit, err := r.Repo.CommitObject(head.Hash())
	if err != nil {
		return "", err
	}

	return commit.Hash.String(), nil
}

func (r Repository) Branch() (string, error) {
	commit, err := r.Commit()
	if err != nil {
		return "", err
	}

	branches, err := r.Repo.Branches()
	if err != nil {
		return "", err
	}

	branch := ""
	err = branches.ForEach(func (ref *plumbing.Reference) error {
		if ref.Hash().String() != commit {
			return nil
		}

		branch = ref.Name().Short()
		return nil
	})
	if err != nil {
		return "", err
	}

	return branch, nil
}

func (r Repository) RepoUrl(remoteName string) (string, error) {
	list, err := r.Repo.Remotes()
	if err != nil {
		return "", err
	}

	for _, r := range list {
		if r.Config().Name == remoteName {
			return r.Config().URLs[0], nil
		}
	}

	return "", errors.New(fmt.Sprintf("Remote \"%s\" not found.", remoteName))
}

func (r Repository) GitEnv() map[string]string {
	tag, _ := r.Tag()
	commit, _ := r.Commit()
	branch, _ := r.Branch()

	shortCommit := ""
	if len(commit) > 8 {
		shortCommit = commit[:8]
	}

	return map[string]string{
		"GIT_TAG": tag,
		"GIT_COMMIT": commit,
		"GIT_COMMIT_SHORT": shortCommit,
		"GIT_BRANCH": branch,
	}
}
