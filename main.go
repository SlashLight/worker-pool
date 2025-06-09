package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

func worker(id int, jobs <-chan string, quit <-chan struct{}, wg *sync.WaitGroup, currentGoroutines *int32) {
	defer wg.Done()

	defer atomic.AddInt32(currentGoroutines, -1)

	for {
		select {
		case <-quit:
			fmt.Printf("Worker %d received quit signal, stopping...\n", id)
			return
		case job, ok := <-jobs:
			if !ok {
				fmt.Printf("Worker %d: job channel closed, exiting...\n", id)
				return
			}
			fmt.Printf("Worker %d processing job: %s\n", id, job)
			time.Sleep(500 * time.Millisecond)
			fmt.Printf("Worker %d finished job: %s\n", id, job)
		}

	}
}

const (
	managerInterval    = 500 * time.Millisecond
	scaleUpThreshold   = 5  // количество задач в очереди для увеличения количества горутин
	scaleDownThreshold = 1  // количество задач в очереди для уменьшения количества горутин
	maxGoroutines      = 20 // максимальное количество горутин-воркеров
	minGoroutines      = 3  // минимальное количество горутин-воркеров
)

func manager(ctx context.Context, jobs <-chan string, quit chan struct{}, wg *sync.WaitGroup, currentGoroutines *int32, workerID *int) {
	ticker := time.NewTicker(managerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			activeGoroutines := atomic.LoadInt32(currentGoroutines)
			jobsInQueue := len(jobs)

			if jobsInQueue > scaleUpThreshold && activeGoroutines < maxGoroutines {
				*workerID++
				atomic.AddInt32(currentGoroutines, 1)

				wg.Add(1)
				go worker(*workerID, jobs, quit, wg, currentGoroutines)
				fmt.Printf("Scaled up: added worker %d, total workers: %d\n", *workerID, activeGoroutines+1)
			} else if jobsInQueue < scaleDownThreshold && activeGoroutines > minGoroutines {
				select {
				case quit <- struct{}{}:
					fmt.Printf("Scaled down: total workers: %d\n", activeGoroutines-1)
				default:
				}
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				fmt.Printf("Manager stopped: %w", err)
				return
			}

			fmt.Println("Manager context done, stopping...")
			return
		}
	}
}

const (
	taskQueueSize  = 10            // размер очереди задач
	initGoroutines = minGoroutines // начальное количество горутин-воркеров
)

func main() {
	// инициализировать канал
	jobs := make(chan string, taskQueueSize)
	wg := &sync.WaitGroup{}
	var currentGoroutines int32 = initGoroutines
	var workerID int = initGoroutines

	//создать начальное количество горутин-воркеров
	quit := make(chan struct{})
	for i := 1; i <= initGoroutines; i++ {
		wg.Add(1)
		go worker(i, jobs, quit, wg, &currentGoroutines)
	}

	// создать горутину-менеджер
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager(ctx, jobs, quit, wg, &currentGoroutines, &workerID)

	// создать задачи для горутин
	var numberOfJobs int
	fmt.Scan(&numberOfJobs)
	go func() {
		for i := 0; i < numberOfJobs; i++ {
			if i%25 == 0 {
				time.Sleep(3 * time.Second)
			}
			job := fmt.Sprintf("Job %d", i+1)
			jobs <- job
		}
		close(jobs)
	}()

	wg.Wait()

}
