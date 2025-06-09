package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

func worker(id int, jobs <-chan string, wg *sync.WaitGroup, currentGoroutines *int32) {
	defer wg.Done()

	defer atomic.AddInt32(currentGoroutines, -1)

	for job := range jobs {
		fmt.Printf("Worker %d processing job: %s\n", id, job)
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("Worker %d finished job: %s\n", id, job)
	}
}

const (
	managerInterval = 500 * time.Millisecond
	scaleUpFactor   = 5  // коэффициент масштабирования при увеличении количества горутин
	maxGoroutines   = 20 // максимальное количество горутин-воркеров
)

func manager(ctx context.Context, jobs <-chan string, wg *sync.WaitGroup, currentGoroutines *int32, workerID *int) {
	ticker := time.NewTicker(managerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			activeGoroutines := atomic.LoadInt32(currentGoroutines)
			jobsInQueue := len(jobs)

			if jobsInQueue > scaleUpFactor && activeGoroutines < maxGoroutines {
				*workerID++
				atomic.AddInt32(currentGoroutines, 1)

				wg.Add(1)
				go worker(*workerID, jobs, wg, currentGoroutines)
				fmt.Printf("Scaled up: added worker %d, total workers: %d\n", *workerID, activeGoroutines+1)
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
	initGoroutines = 3  // начальное количество горутин-воркеров
	taskQueueSize  = 10 // размер очереди задач
)

func main() {
	// инициализировать канал
	jobs := make(chan string, taskQueueSize)
	wg := &sync.WaitGroup{}
	var currentGoroutines int32 = initGoroutines
	var workerID int = initGoroutines

	//создать начальное количество горутин-воркеров
	for i := 1; i <= initGoroutines; i++ {
		wg.Add(1)
		go worker(i, jobs, wg, &currentGoroutines)
	}

	// создать горутину-менеджер
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager(ctx, jobs, wg, &currentGoroutines, &workerID)

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
