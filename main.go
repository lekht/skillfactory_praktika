package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const buffSize int = 10
const bufferEmptyingInterval time.Duration = 5 * time.Second

type RingBuffer struct {
	buf    []int
	size   int
	w      int // next position to write
	isFull bool
	mu     sync.Mutex
}

// Return a point to a new ring buffer
func NewBuff(size int) *RingBuffer {
	return &RingBuffer{
		buf:    make([]int, size),
		size:   buffSize,
		w:      0,
		isFull: false,
		mu:     sync.Mutex{},
	}
}

//Read a console input
func (r *RingBuffer) Init() (chan int, <-chan interface{}) {
	output := make(chan int)
	done := make(chan interface{})
	go func() {
		defer close(output)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanWords)
		var text string
		for scanner.Scan() {
			text = scanner.Text()
			if strings.EqualFold(text, "exit") {
				log.Fatal("======\nЗавершение работы\n======")
			}
			i, err := strconv.Atoi(text)
			if err != nil {
				log.Println("*****Программа обрабатывает только целые числа!*****")
				continue
			}
			if err != nil {
				fmt.Println(err.Error())
			}
			output <- i
			log.Printf("Число %d принято в работу", i)
		}

	}()
	return output, done
}

//Push number to the ring buffer and do nothing if buffer is full
func (r *RingBuffer) Push(i int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isFull {
		log.Println("Буффер полон")
		return
	}
	r.buf[r.w] = i
	r.w++
	log.Printf("Число %d добавлено в буфер", i)
	if r.w >= r.size {
		r.isFull = true
	}
}

//Take one number from buffer and put it in the channel for consumer
func (r *RingBuffer) Get(output chan int) {
	if r.w <= 0 {
		return
	}
	output <- r.buf[0]
	for i := 0; i < r.w-1; i++ {
		r.buf[i] = r.buf[i+1]
	}
	r.w--
	r.isFull = false
}

// Consumer
func Printer(done <-chan interface{}, c <-chan int) {
	for data := range c {
		log.Println("Числа прошедшие фильтрацию: ", data)
	}
}

//The negative numbers filtration
func NegativeNumbFilter(done <-chan interface{}, c <-chan int) <-chan int {
	output := make(chan int)

	go func() {
		for {
			select {
			case i := <-c:
				if i >= 0 {
					output <- i
				}
			case <-done:
				return
			}
		}

	}()

	return output
}

// The special filtration's stage
func UserFilter(done <-chan interface{}, c <-chan int) <-chan int {
	output := make(chan int)

	go func() {
		for {
			select {
			case i := <-c:
				if i != 0 && i%3 == 0 {
					output <- i
				}
			case <-done:
				close(output)
				return
			}
		}
	}()

	return output
}

// Push data to the ring buffer after all stages
func (r *RingBuffer) BufferStage(done <-chan interface{}, c <-chan int) <-chan int {
	filtered := make(chan int)

	go func() {
		for {
			select {
			case i := <-c:
				r.Push(i)
			case <-done:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-time.After(bufferEmptyingInterval):
				r.Get(filtered)
			case <-done:
				return
			}

		}
	}()

	return filtered
}

func main() {
	ring := NewBuff(buffSize)
	data, done := ring.Init()
	nonNegativeInt := NegativeNumbFilter(done, data)
	userFilteredInt := UserFilter(done, nonNegativeInt)
	forConsumer := ring.BufferStage(done, userFilteredInt)
	Printer(done, forConsumer)
}
