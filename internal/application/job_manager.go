package application

import (
	"context"
	"sync"
)

// JobManager は 1 本のワーカー向けにジョブ ID を順番に渡す。
type JobManager struct {
	mu           sync.Mutex
	queue        []string
	notify       chan struct{}
	activeJob    string
	activeCancel context.CancelFunc
}

// NewJobManager は空のキューを作る。
func NewJobManager() *JobManager {
	return &JobManager{
		notify: make(chan struct{}, 1),
	}
}

// Enqueue は待機キューへ ID を追加する。
func (m *JobManager) Enqueue(jobID string) {
	m.mu.Lock()
	m.queue = append(m.queue, jobID)
	m.mu.Unlock()
	m.signal()
}

// Dequeue は次の ID を返す。キューが空なら ctx が終わるまで待つ。
func (m *JobManager) Dequeue(ctx context.Context) (string, bool) {
	for {
		m.mu.Lock()
		if len(m.queue) > 0 {
			id := m.queue[0]
			m.queue = m.queue[1:]
			m.mu.Unlock()
			return id, true
		}
		m.mu.Unlock()

		select {
		case <-ctx.Done():
			return "", false
		case <-m.notify:
		}
	}
}

// RemoveQueued は待機中の ID をキューから取り除く。見つかったら true。
func (m *JobManager) RemoveQueued(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, id := range m.queue {
		if id == jobID {
			m.queue = append(m.queue[:i], m.queue[i+1:]...)
			return true
		}
	}
	return false
}

// SetActive は実行中ジョブと取消関数を記録する。
func (m *JobManager) SetActive(jobID string, cancel context.CancelFunc) {
	m.mu.Lock()
	m.activeJob = jobID
	m.activeCancel = cancel
	m.mu.Unlock()
}

// ClearActive は実行中ジョブの記録を消す。
func (m *JobManager) ClearActive() {
	m.mu.Lock()
	m.activeJob = ""
	m.activeCancel = nil
	m.mu.Unlock()
}

// CancelActive は実行中ジョブの context を取消する。
func (m *JobManager) CancelActive(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeJob != jobID || m.activeCancel == nil {
		return false
	}
	m.activeCancel()
	return true
}

// ActiveJobID は実行中のジョブ ID を返す。無ければ空文字。
func (m *JobManager) ActiveJobID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeJob
}

func (m *JobManager) signal() {
	select {
	case m.notify <- struct{}{}:
	default:
	}
}
