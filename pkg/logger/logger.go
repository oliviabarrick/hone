package logger

import (
	"log"
	"github.com/justinbarrick/farm/pkg/config"
)

func LogJob(callback func (config.Job) error) func (config.Job) error {
	return func (job config.Job) error {
		log.Printf("===> Running job \"%s\".\n", job.Name)
		err := callback(job)
		if err != nil {
			log.Printf("===> Job \"%s\" errored: %s.\n", job.Name, err)
		} else {
			log.Printf("===> Job \"%s\" completed!\n", job.Name)
		}
		return err
	}
}
