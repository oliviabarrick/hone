package scm

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/h2non/gock"
)

func TestSCM(t *testing.T) {
	defer gock.Off()

	repo := "justinbarrick/hone"

	gock.New("https://api.github.com").
		Post("/repos/justinbarrick/hone/statuses/43f77731a7a882a2616cfadcd9d23cd723f871ab").
		Reply(200).
		Type("application/json").JSON(map[string]string{})

	scm := SCM{
		Token: "API_TOKEN",
		Repo: &repo,
	}

	err := scm.Init(context.TODO())
	assert.Nil(t, err)
	err = scm.PostStatus(StatePending, "43f77731a7a882a2616cfadcd9d23cd723f871ab", "success", "")
	assert.Nil(t, err)
}
