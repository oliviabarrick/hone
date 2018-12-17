package scm

import (
	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/gitea"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/gogs"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/drone/go-scm/scm/transport"
	"github.com/justinbarrick/hone/pkg/git"
	"github.com/justinbarrick/hone/pkg/events"
	"github.com/justinbarrick/hone/pkg/logger"
	"context"
	"errors"
	"net/http"
)

type State int

const (
	StateUnknown State = iota
	StatePending
	StateRunning
	StateSuccess
	StateFailure
	StateCanceled
	StateError
)

type Provider string

const (
	ProviderGithub    Provider = "github"
	ProviderBitbucket Provider = "bitbucket"
	ProviderGitlab    Provider = "gitlab"
	ProviderGitea     Provider = "gitea"
	ProviderGogs      Provider = "gogs"
	ProviderStash     Provider = "stash"
)

type SCM struct {
	Provider *Provider  `hcl:"provider"`
	URL      *string    `hcl:"url"`
	Token    string     `hcl:"token"`
	Repo     *string    `hcl:"repo"`
	Remote   *string    `hcl:"remote"`
	Condition *string   `hcl:"condition"`
	Git      git.Repository
	commit   string
	client   *scm.Client
	ctx      context.Context
}

func (s *SCM) GetURL() (string, error) {
	defaultURL := map[Provider]string{
		ProviderGithub:    "https://api.github.com/",
		ProviderBitbucket: "https://api.bitbucket.org/",
		ProviderGitlab:    "https://gitlab.com/",
	}

	if s.URL != nil {
		return *s.URL, nil
	}

	provider := s.GetProvider()

	if defaultURL[provider] == "" {
		return "", errors.New("URL must be provided for selected SCM provider")
	}

	return defaultURL[provider], nil
}

func (s *SCM) GetProvider() (Provider) {
	urlToProvider := map[string]Provider {
		"github.com": ProviderGithub,
		"bitbucket.com": ProviderBitbucket,
		"bitbucket.org": ProviderBitbucket,
		"gitlab.com": ProviderGitlab,
	}

	var provider Provider
	if s.Provider != nil {
		provider = *s.Provider
	} else {
		remote := "origin"
		if s.Remote != nil {
			remote = *s.Remote
		}
		repo, _ := s.Git.RepoHostname(remote)
		provider = urlToProvider[repo]
	}

	if provider == Provider("") {
		provider = ProviderGithub
	}

	return provider
}

func (s *SCM) GetRepo() string {
	var repo string
	if s.Repo != nil {
		repo = *s.Repo
	} else {
		remote := "origin"
		if s.Remote != nil {
			remote = *s.Remote
		}
		repo, _ = s.Git.RepoPath(remote)
	}

	return repo
}

func (s *SCM) Init(ctx context.Context) (err error) {
	repo, err := git.NewRepository()
	if err != nil {
		return err
	}
	s.Git = repo

	s.commit, err = s.Git.Commit()
	if err != nil {
		return err
	}

	url, err := s.GetURL()
	if err != nil {
		return
	}

	switch s.GetProvider() {
	case ProviderGithub:
		s.client, err = github.New(url)
	case ProviderBitbucket:
		s.client, err = bitbucket.New(url)
	case ProviderGitlab:
		s.client, err = gitlab.New(url)
	case ProviderGitea:
		s.client, err = gitea.New(url)
	case ProviderGogs:
		s.client, err = gogs.New(url)
	case ProviderStash:
		s.client, err = stash.New(url)
	default:
		return errors.New("Unknown SCM provider.")
	}

	if err != nil {
		return
	}

	s.client.Client = &http.Client{
		Transport: &transport.BearerToken{
			Token: s.Token,
		},
	}

	s.ctx = ctx
	return
}

func (s SCM) PostStatus(state State, commit string, message string) error {
	status := &scm.StatusInput{
		State: scm.State(state),
		Label: "hone",
		Desc:  message,
	}

	_, _, err := s.client.Repositories.CreateStatus(s.ctx, s.GetRepo(), commit, status)
	return err
}

func (s SCM) BuildStarted() error {
	return s.PostStatus(StateRunning, s.commit, "Build started!")
}

func (s SCM) BuildCompleted() error {
	return s.PostStatus(StateSuccess, s.commit, "Build completed successfully!")
}

func (s SCM) BuildFailed() error {
	return s.PostStatus(StatePending, s.commit, "Build failed!")
}

func (s SCM) BuildErrored() error {
	return s.PostStatus(StateError, s.commit, "Build errored due to a configuration error!")
}

func (s SCM) BuildCanceled() error {
	return s.PostStatus(StateCanceled, s.commit, "Build cancelled by user!")
}

func InitSCMs(scms []*SCM, env map[string]interface{}) ([]*SCM, error) {
	finalScms := []*SCM{}

	// TODO: Status() doesn't ignore gitignore: https://github.com/src-d/go-git/issues/844
	/*
	repo, err := git.NewRepository()
	if err != nil {
		return finalScms, err
	}

	dirty, err := repo.IsDirty()
	if err != nil {
		return finalScms, errors.New(fmt.Sprintf("checking if repository is dirty: %s", err))
	} else if dirty {
		logger.Printf("Not posting status because directory is dirty.\n")
		return finalScms, nil
	}
	*/

	for _, scm := range scms {
		run, err := events.YQLMatch(scm.Condition, env)
		if err != nil {
			return finalScms, err
		}

		if run == false || scm.Token == "" {
			continue
		}

		err = scm.Init(context.TODO())
		if err != nil {
			return finalScms, err
		}

		logger.Printf("Initialized reporting provider: %s\n", scm.GetProvider())
		finalScms = append(finalScms, scm)
	}

	return finalScms, nil
}