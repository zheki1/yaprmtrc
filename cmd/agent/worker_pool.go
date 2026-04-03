package main

import "log"

// Job — единица работы, выполняемая воркером пула.
type Job func() error

// StartWorkers запускает n воркеров, читающих задачи из канала jobs.
func StartWorkers(n int, jobs <-chan Job) {
	if n < 1 {
		n = 1
	}

	for i := 0; i < n; i++ {
		go func(id int) {
			for job := range jobs {
				if job == nil {
					continue
				}

				if err := job(); err != nil {
					log.Printf("Job failed %s %v \n", err, id)
				}
			}
		}(i)
	}
}
