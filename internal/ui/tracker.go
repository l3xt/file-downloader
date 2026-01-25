package ui

import (
	"fmt"
	"io"
	"sync"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type Tracker interface {
	Logf(format string, args ...interface{})
	ProxyReader(r io.Reader) io.ReadCloser
	SetCurrent(current int64)
	SetTotal(total int64)
}

type Container struct {
	progress *mpb.Progress
}

// Структура ContainerBar реализует интерфейс Tracker
type ContainerBar struct {
	c   *Container
	bar *mpb.Bar
}

func New(wg *sync.WaitGroup) *Container {
	return &Container{progress: mpb.New(mpb.WithWaitGroup(wg))}
}

func (c *Container) Wait() {
	c.progress.Wait()
}

func (c *Container) Logf(format string, args ...any) {
	c.progress.Write([]byte(fmt.Sprintf(format, args...) + "\n"))
}

func (c *Container) AddBar(total int64, name string) *ContainerBar {
	bar := c.progress.AddBar(total,
		mpb.PrependDecorators(
			decor.Name(name),
		),
		mpb.AppendDecorators(
			decor.Percentage(),
			decor.CountersKibiByte(" %.2f / %.2f"),
		),
	)
	return &ContainerBar{
		c:   c,
		bar: bar,
	}
}

func (cb *ContainerBar) Logf(format string, args ...any) {
	cb.c.Logf(format, args...)
}

func (cb *ContainerBar) ProxyReader(r io.Reader) io.ReadCloser {
	return cb.bar.ProxyReader(r)
}

func (cb *ContainerBar) SetTotal(total int64) {
	cb.bar.SetTotal(total, false)
	cb.bar.EnableTriggerComplete()
}

func (cb *ContainerBar) SetCurrent(current int64) {
	cb.bar.SetCurrent(current)
}