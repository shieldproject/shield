// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package bufpipe_test

import "io"
import "fmt"
import "time"
import "sync"
import "math/rand"
import "github.com/dsnet/golib/bufpipe"

func randomChars(cnt int, rand *rand.Rand) string {
	data := make([]byte, cnt)
	for idx := range data {
		char := byte(rand.Intn(10 + 26 + 26))
		if char < 10 {
			data[idx] = '0' + char
		} else if char < 10+26 {
			data[idx] = 'A' + char - 10
		} else {
			data[idx] = 'a' + char - 36
		}
	}
	return string(data)
}

// In LineMono mode, the consumer cannot see the written data until the pipe is
// closed. Thus, it is possible for the producer to go back to the front of the
// pipe and record the total number of bytes written out. This functionality is
// useful in cases where a file format's header contains information that is
// dependent on what is eventually written.
func ExampleBufferPipe_lineMono() {
	// The buffer is small enough such that the producer does hit the limit.
	buffer := bufpipe.NewBufferPipe(make([]byte, 256), bufpipe.LineMono)

	rand := rand.New(rand.NewSource(0))
	group := new(sync.WaitGroup)
	group.Add(2)

	// Producer routine.
	go func() {
		defer group.Done()
		defer buffer.Close()

		// In LineMono mode only, it is safe to store a reference to written
		// data and modify later.
		header, _, err := buffer.WriteSlices()
		if err != nil {
			panic(err)
		}

		totalCnt, _ := buffer.Write([]byte("#### "))
		for idx := 0; idx < 8; idx++ {
			data := randomChars(rand.Intn(64), rand) + "\n"

			// So long as the amount of data written has not exceeded the size
			// of the buffer, Write will never fail.
			cnt, err := buffer.Write([]byte(data))
			totalCnt += cnt
			if err == io.ErrShortWrite {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		// Write the header afterwards
		copy(header[:4], fmt.Sprintf("%04d", totalCnt))
	}()

	// Consumer routine.
	go func() {
		defer group.Done()

		// In LineMono mode only, a call to ReadSlices is guaranteed to block
		// until the channel is closed. All written data will be made available.
		data, _, _ := buffer.ReadSlices()
		buffer.ReadMark(len(data)) // Technically, this is optional

		fmt.Println(string(data))
	}()

	group.Wait()

	// Output:
	// 0256 kdUhQzHYs2LjaukXEC292UgLOCAPQTCNAKfc0XMNCUuJbsqiHmm6GJMFck
	// whxMYR1k
	// zhMYzktxIv10mIPqBCCwm646E6chwIFZfpX0fjqMu0YKLDhfIMnDq8w9J
	// fQhkT1qEkJfEI0jtbDnIrEXx6G4xMgXEB6auAyBUjPk2jMSgCMVZf8L1VgJemin
	// 2Quy1C5aA00KbYqawNeuXYTvgeUXGu3zyjMUoEIrOx7
	// ecE4dY3ZaTrX03xBY
}

// In LineDual mode, the consumer sees produced data immediately as it becomes
// available. The producer is only allowed to write as much data as the size of
// the underlying buffer. The amount that can be written is independent of the
// operation of the consumer.
func ExampleBufferPipe_lineDual() {
	// The buffer is small enough such that the producer does hit the limit.
	buffer := bufpipe.NewBufferPipe(make([]byte, 256), bufpipe.LineDual)

	rand := rand.New(rand.NewSource(0))
	group := new(sync.WaitGroup)
	group.Add(2)

	// Producer routine.
	go func() {
		defer group.Done()
		defer buffer.Close()

		buffer.Write([]byte("#### ")) // Write a fake header
		for idx := 0; idx < 8; idx++ {
			data := randomChars(rand.Intn(64), rand) + "\n"

			// So long as the amount of data written has not exceeded the size
			// of the buffer, Write will never fail.
			if _, err := buffer.Write([]byte(data)); err == io.ErrShortWrite {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Consumer routine.
	go func() {
		defer group.Done()
		for {
			// Reading can be also done using ReadSlices and ReadMark pairs.
			data, _, err := buffer.ReadSlices()
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
			buffer.ReadMark(len(data))
			fmt.Print(string(data))
		}
		fmt.Println()
	}()

	group.Wait()

	// Output:
	// #### kdUhQzHYs2LjaukXEC292UgLOCAPQTCNAKfc0XMNCUuJbsqiHmm6GJMFck
	// whxMYR1k
	// zhMYzktxIv10mIPqBCCwm646E6chwIFZfpX0fjqMu0YKLDhfIMnDq8w9J
	// fQhkT1qEkJfEI0jtbDnIrEXx6G4xMgXEB6auAyBUjPk2jMSgCMVZf8L1VgJemin
	// 2Quy1C5aA00KbYqawNeuXYTvgeUXGu3zyjMUoEIrOx7
	// ecE4dY3ZaTrX03xBY
}

// In RingBlock mode, the consumer sees produced data immediately as it becomes
// available. The producer is allowed to write as much data as it wants so long
// as the consumer continues to read the data in the pipe.
func ExampleBufferPipe_ringBlock() {
	// Intentionally small buffer to show that data written into the buffer
	// can exceed the size of the buffer itself.
	buffer := bufpipe.NewBufferPipe(make([]byte, 64), bufpipe.RingBlock)

	rand := rand.New(rand.NewSource(0))
	group := new(sync.WaitGroup)
	group.Add(2)

	// Producer routine.
	go func() {
		defer group.Done()
		defer buffer.Close()

		buffer.Write([]byte("#### ")) // Write a fake header
		for idx := 0; idx < 8; idx++ {
			data := randomChars(rand.Intn(64), rand) + "\n"

			// So long as the amount of data written has not exceeded the size
			// of the buffer, Write will never fail.
			buffer.Write([]byte(data))

			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Consumer routine.
	go func() {
		defer group.Done()

		data := make([]byte, 64)
		for {
			// Reading can also be done using the Read method.
			cnt, err := buffer.Read(data)
			fmt.Print(string(data[:cnt]))
			if err == io.EOF {
				break
			}
		}
		fmt.Println()
	}()

	group.Wait()

	// Output:
	// #### kdUhQzHYs2LjaukXEC292UgLOCAPQTCNAKfc0XMNCUuJbsqiHmm6GJMFck
	// whxMYR1k
	// zhMYzktxIv10mIPqBCCwm646E6chwIFZfpX0fjqMu0YKLDhfIMnDq8w9J
	// fQhkT1qEkJfEI0jtbDnIrEXx6G4xMgXEB6auAyBUjPk2jMSgCMVZf8L1VgJemin
	// 2Quy1C5aA00KbYqawNeuXYTvgeUXGu3zyjMUoEIrOx7
	// ecE4dY3ZaTrX03xBYJ04OzomME36yth76CFmg2zTolzKhYByvZ8
	// FQMuYbcWHLcUu4yL3aBZkwJrbDFUcHpGnBGfbDq4aFlLS5vGOm6mYOjHZll
	// iP0QQKpKp3cz
}
