package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

func worker(id int32, jobs <-chan string, quit <-chan struct{}, wg *sync.WaitGroup, currentGoroutines *int32) {
	defer wg.Done()

	for {
		select {
		case <-quit:
			fmt.Printf("Worker %d received quit signal, stopping...\n", id)
			atomic.AddInt32(currentGoroutines, -1)
			return
		case job, ok := <-jobs:
			if !ok {
				fmt.Printf("Worker %d job channel closed, exiting...\n", id)
				atomic.AddInt32(currentGoroutines, -1)
				return
			}
			fmt.Printf("Worker %d processing job: %s\n", id, job)
			time.Sleep(500 * time.Millisecond)
			fmt.Printf("Worker %d finished job: %s\n", id, job)
		}

	}
}

const (
	managerInterval = 500 * time.Millisecond

	scaleUpThreshold   = 5 // количество задач в очереди для увеличения количества горутин
	scaleDownThreshold = 1 // количество задач в очереди для уменьшения количества горутин

	maxGoroutines = 20 // максимальное количество горутин-воркеров
	minGoroutines = 3  // минимальное количество горутин-воркеров
)

func manager(ctx context.Context, jobs <-chan string, quit chan struct{}, wg *sync.WaitGroup, currentGoroutines *int32, lastWorkerID *int32) {
	ticker := time.NewTicker(managerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			activeGoroutines := atomic.LoadInt32(currentGoroutines)
			jobsInQueue := len(jobs)

			if jobsInQueue > scaleUpThreshold && activeGoroutines < maxGoroutines {
				newID := atomic.AddInt32(lastWorkerID, 1)
				newCount := atomic.AddInt32(currentGoroutines, 1)

				wg.Add(1)
				go worker(newID, jobs, quit, wg, currentGoroutines)
				fmt.Printf("Scaled up: added worker %d, total workers: %d\n", newID, newCount)
				continue
			}

			if jobsInQueue < scaleDownThreshold && activeGoroutines > minGoroutines {
				select {
				case quit <- struct{}{}:
					fmt.Printf("Scaled down: total workers: %d\n", activeGoroutines-1)
				default:
				}
			}

		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				fmt.Printf("Manager stopped: %v\n", err)
				return
			}

			fmt.Println("Manager context done, stopping...")
			return
		}
	}
}

const (
	// Настроим сон для генератора, чтобы имитировать неравномерную нагрузку задачами
	generatorSleepInterval  = 3 * time.Second // интервал между генерацией задач
	generatorSleepFrequency = 25              // частота, с которой горутина-генератор будет "спать"
)

func taskGenerator(jobs chan<- string, numberOfJobs int, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < numberOfJobs; i++ {
		if i%generatorSleepFrequency == 0 {
			time.Sleep(generatorSleepInterval)
		}
		job := fmt.Sprintf("Job %d", i+1)
		jobs <- job
	}
	close(jobs)
}

const (
	taskQueueSize  = 10            // размер очереди задач
	initGoroutines = minGoroutines // начальное количество горутин-воркеров
)

func main() {
	jobs := make(chan string, taskQueueSize)
	var (
		wg                sync.WaitGroup
		currentGoroutines int32 = initGoroutines
		lastWorkerID      int32 = initGoroutines
	)

	quit := make(chan struct{})
	for i := 1; i <= initGoroutines; i++ {
		wg.Add(1)
		go worker(int32(i), jobs, quit, &wg, &currentGoroutines)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager(ctx, jobs, quit, &wg, &currentGoroutines, &lastWorkerID)

	var (
		numberOfJobs int
		generatorWg  sync.WaitGroup
	)
	fmt.Print("Enter the number of tasks: ")
	fmt.Scan(&numberOfJobs)

	generatorWg.Add(1)
	go taskGenerator(jobs, numberOfJobs, &generatorWg)

	generatorWg.Wait()
	wg.Wait()
}
