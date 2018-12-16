package events

import (
	"fmt"
	"github.com/justinbarrick/hone/pkg/job"
	"github.com/justinbarrick/hone/pkg/config/types"
	"github.com/justinbarrick/hone/pkg/logger"
	"github.com/caibirdme/yql"
)

func EventCallback(config *types.Config, cb func (j *job.Job) error) (func (j *job.Job) error) {
	return func (j *job.Job) error {
		if j.Condition == nil {
			return cb(j)
		}

		result, err := yql.Match(*j.Condition, config.Env)
		if err != nil {
			return err
		}

		if result == true {
			return cb(j)
		}


		logger.Log(j, fmt.Sprintf("Skipping job, since condition not met: %s", *j.Condition))
		return nil
	}
}
