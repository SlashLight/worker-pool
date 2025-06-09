package main

import (
	"fmt"
	"sync"
	"time"
)

func worker(id int, jobs <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		fmt.Printf("Worker %d processing job: %s\n", id, job)
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("Worker %d finished job: %s\n", id, job)
	}
}

const (
	initGoroutines = 3 // начальное количество горутин-воркеров
)

func main() {
	// инициализировать канал
	jobs := make(chan string)

	wg := &sync.WaitGroup{}

	//создать начальное количество горутин-воркеров
	for i := 1; i <= initGoroutines; i++ {
		wg.Add(1)
		go worker(i, jobs, wg)
	}

	// создать задачи для горутин
	var numberOfJobs int
	fmt.Scan(&numberOfJobs)
	go func() {
		for i := 0; i < numberOfJobs; i++ {
			job := fmt.Sprintf("Job %d", i+1)
			jobs <- job
		}
		close(jobs)
	}()

	wg.Wait()

}
