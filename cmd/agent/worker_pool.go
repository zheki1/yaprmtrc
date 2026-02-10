package main

import "fmt"

type Job func() error

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
					fmt.Printf("Job failed %s %v \n", err, id)
				}
			}
		}(i)
	}
}
