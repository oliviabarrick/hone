package git

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-git.v4"
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

func RepoUrl(remoteName string) (string, error) {
	r, err := git.PlainOpen(".")
	if err != nil {
		return "", err
	}

	list, err := r.Remotes()
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
