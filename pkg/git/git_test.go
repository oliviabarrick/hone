package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func emptyRepo(t *testing.T) *git.Repository {
	r, err := git.Init(memory.NewStorage(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func doCommit(t *testing.T, r *git.Repository) plumbing.Hash {
	tree, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	hash, err := tree.Commit("First commit!", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "test@test.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func oneCommit(t *testing.T) *git.Repository {
	fs := memfs.New()

	r, err := git.Init(memory.NewStorage(), fs)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	fs.Create("my-file")
	tree.Add("my-file")

	doCommit(t, r)
	return r
}

func noBranch(t *testing.T) *git.Repository {
	r := oneCommit(t)

	tree, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	_ = doCommit(t, r)
	hash := doCommit(t, r)
	_ = doCommit(t, r)

	err = tree.Checkout(&git.CheckoutOptions{
		Hash: hash,
	})
	if err != nil {
		t.Fatal(err)
	}

	return r
}

func makeBranch(t *testing.T) *git.Repository {
	r := oneCommit(t)

	tree, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	err = tree.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.ReferenceName("refs/heads/my-branch"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = doCommit(t, r)

	return r
}

func taggedCommit(t *testing.T) *git.Repository {
	r := oneCommit(t)
	hash := doCommit(t, r)

	_, err := r.CreateTag("my-tag", hash, nil)
	if err != nil {
		t.Fatal(err)
	}

	return r
}

func TestGitTagRepoNotFound(t *testing.T) {
	r := Repository{
		Repo: nil,
	}

	tag, err := r.Tag()
	assert.Equal(t, err.Error(), "Repository not set.")
	assert.Equal(t, tag, "")
}

func TestGitTagRepoEmptyRepo(t *testing.T) {
	r := Repository{
		Repo: emptyRepo(t),
	}

	tag, err := r.Tag()
	assert.Equal(t, err.Error(), "reference not found")
	assert.Equal(t, tag, "")
}

func TestGitTagRepoNoTag(t *testing.T) {
	r := Repository{
		Repo: oneCommit(t),
	}

	tag, err := r.Tag()
	assert.Nil(t, err)
	assert.Equal(t, tag, "")
}

func TestGitTagRepoTagged(t *testing.T) {
	r := Repository{
		Repo: taggedCommit(t),
	}

	tag, err := r.Tag()
	assert.Nil(t, err)
	assert.Equal(t, tag, "my-tag")
}

func TestGitCommitRepoNotFound(t *testing.T) {
	r := Repository{
		Repo: nil,
	}

	commit, err := r.Commit()
	assert.Equal(t, err.Error(), "Repository not set.")
	assert.Equal(t, commit, "")
}

func TestGitCommitRepoEmptyRepo(t *testing.T) {
	r := Repository{
		Repo: emptyRepo(t),
	}

	commit, err := r.Commit()
	assert.Equal(t, err.Error(), "reference not found")
	assert.Equal(t, commit, "")
}

func TestGitCommitRepoOneCommit(t *testing.T) {
	r := Repository{
		Repo: oneCommit(t),
	}

	commit, err := r.Commit()
	assert.Nil(t, err)
	assert.Equal(t, commit, "1fb2434172f86f213dc75ae4c838264f5d9bfb32")
}

func TestGitBranchRepoNotFound(t *testing.T) {
	r := Repository{
		Repo: nil,
	}

	branch, err := r.Branch()
	assert.Equal(t, err.Error(), "Repository not set.")
	assert.Equal(t, branch, "")
}

func TestGitBranchRepoEmptyRepo(t *testing.T) {
	r := Repository{
		Repo: emptyRepo(t),
	}

	branch, err := r.Branch()
	assert.Equal(t, err.Error(), "reference not found")
	assert.Equal(t, branch, "")
}

func TestGitBranchRepoNoBranch(t *testing.T) {
	r := Repository{
		Repo: noBranch(t),
	}

	branch, err := r.Branch()
	assert.Nil(t, err)
	assert.Equal(t, branch, "")
}

func TestGitBranch(t *testing.T) {
	r := Repository{
		Repo: makeBranch(t),
	}

	branch, err := r.Branch()
	assert.Nil(t, err)
	assert.Equal(t, branch, "my-branch")
}

func TestGitGitEnvRepoNotFound(t *testing.T) {
	r := Repository{
		Repo: nil,
	}

	gitEnv := r.GitEnv()
	assert.Equal(t, gitEnv["GIT_TAG"], "")
	assert.Equal(t, gitEnv["GIT_COMMIT"], "")
	assert.Equal(t, gitEnv["GIT_COMMIT_SHORT"], "")
	assert.Equal(t, gitEnv["GIT_BRANCH"], "")

}

func TestGitGitEnvRepoEmptyRepo(t *testing.T) {
	r := Repository{
		Repo: emptyRepo(t),
	}

	gitEnv := r.GitEnv()
	assert.Equal(t, gitEnv["GIT_TAG"], "")
	assert.Equal(t, gitEnv["GIT_COMMIT"], "")
	assert.Equal(t, gitEnv["GIT_COMMIT_SHORT"], "")
	assert.Equal(t, gitEnv["GIT_BRANCH"], "")

}

func TestGitGitEnvRepoNoBranch(t *testing.T) {
	r := Repository{
		Repo: noBranch(t),
	}

	gitEnv := r.GitEnv()
	assert.Equal(t, gitEnv["GIT_TAG"], "")
	assert.Equal(t, gitEnv["GIT_COMMIT"], "45f7c4bc1e422d450f791b1ebe844866dd6f837f")
	assert.Equal(t, gitEnv["GIT_COMMIT_SHORT"], "45f7c4bc")
	assert.Equal(t, gitEnv["GIT_BRANCH"], "")
}

func TestGitGitEnvBranch(t *testing.T) {
	r := Repository{
		Repo: makeBranch(t),
	}

	gitEnv := r.GitEnv()
	assert.Equal(t, gitEnv["GIT_TAG"], "")
	assert.Equal(t, gitEnv["GIT_COMMIT"], "729ffe8860eacb4aa5aaff19e9a05ab6d8cc5ede")
	assert.Equal(t, gitEnv["GIT_COMMIT_SHORT"], "729ffe88")
	assert.Equal(t, gitEnv["GIT_BRANCH"], "my-branch")
}

func TestGitGitEnvTag(t *testing.T) {
	r := Repository{
		Repo: taggedCommit(t),
	}

	gitEnv := r.GitEnv()
	assert.Equal(t, gitEnv["GIT_TAG"], "my-tag")
	assert.Equal(t, gitEnv["GIT_COMMIT"], "729ffe8860eacb4aa5aaff19e9a05ab6d8cc5ede")
	assert.Equal(t, gitEnv["GIT_COMMIT_SHORT"], "729ffe88")
	assert.Equal(t, gitEnv["GIT_BRANCH"], "master")
}

func TestGitRepoUrlNoRepo(t *testing.T) {
	r := Repository{
		Repo: nil,
	}

	repoUrl, err := r.RepoUrl("origin")
	assert.Equal(t, err.Error(), "Repo not set.")
	assert.Equal(t, repoUrl, "")
}

func TestGitRepoUrlRemoteNotFound(t *testing.T) {
	r := Repository{
		Repo: emptyRepo(t),
	}

	repoUrl, err := r.RepoUrl("origin")
	assert.Equal(t, err.Error(), `Remote "origin" not found.`)
	assert.Equal(t, repoUrl, "")
}

func TestGitRepoUrlRemoteFound(t *testing.T) {
	for _, repoUrl := range []string{
		"ssh://github.com/justinbarrick/hone",
		"ssh://github.com:22/justinbarrick/hone",
		"ssh://git@github.com/justinbarrick/hone",
		"ssh://git@github.com:22/justinbarrick/hone",
		"ssh://github.com/justinbarrick/hone.git",
		"ssh://github.com:22/justinbarrick/hone.git",
		"ssh://git@github.com/justinbarrick/hone.git",
		"ssh://git@github.com:22/justinbarrick/hone.git",
		"https://github.com/justinbarrick/hone.git",
		"https://github.com:22/justinbarrick/hone.git",
		"https://git@github.com/justinbarrick/hone.git",
		"https://git@github.com:22/justinbarrick/hone.git",
		"https://github.com/justinbarrick/hone",
		"https://github.com:22/justinbarrick/hone",
		"https://git@github.com/justinbarrick/hone",
		"https://git@github.com:22/justinbarrick/hone",
		"git@github.com:justinbarrick/hone.git",
		"github.com:justinbarrick/hone.git",
		"git@github.com:justinbarrick/hone",
		"github.com:justinbarrick/hone",
	} {
		repo := oneCommit(t)
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{
				repoUrl,
			},
		})

		r := Repository{
			Repo: repo,
		}

		repoUrlParsed, err := r.RepoUrl("origin")
		assert.Nil(t, err)
		assert.Equal(t, repoUrlParsed, repoUrl)
		hostname, err := r.RepoHostname("origin")
		assert.Nil(t, err)
		assert.Equal(t, "github.com", hostname)
		path, err := r.RepoPath("origin")
		assert.Nil(t, err)
		assert.Equal(t, "justinbarrick/hone", path)

	}
}
