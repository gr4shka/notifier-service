package notifierservice

import (
	"sync/atomic"
)

type Stats struct {
	Sent    int64 // успешно отправлено
	Failed  int64 // завершилось ошибкой
	Retries int64 // количество повторных попыток
}

func (s *Stats) IncSent() {
	atomic.AddInt64(&s.Sent, 1)
}

func (s *Stats) IncFailed() {
	atomic.AddInt64(&s.Failed, 1)
}

func (s *Stats) IncRetries() {
	atomic.AddInt64(&s.Retries, 1)
}

func (s *Stats) Snapshot() Stats {
	return Stats{
		Sent:    atomic.LoadInt64(&s.Sent),
		Failed:  atomic.LoadInt64(&s.Failed),
		Retries: atomic.LoadInt64(&s.Retries),
	}
}
