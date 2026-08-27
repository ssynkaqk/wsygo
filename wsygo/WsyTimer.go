package Wsy

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// WsyTimer 调度入口：Timer.Set(...).Set(...).New()
type WsyTimer struct {
	pending []timerTask
	tasks   []*timerTask
	mu      sync.RWMutex
}

// TimerValid 任务配置，用于 Timer.Set("名称", TimerValid{Every: 30, ...})
//
//	Every  循环间隔（秒）；Run 结束且 After 等待后开始倒计时
//	Before 倒计时到 0 后，再等待秒数才执行 Run；0=到点立刻执行
//	After  Run 完成后等待秒数，再开始下一轮 Every；0=不等待
//	Once   true=只执行一次
//	With   绝对时间窗：开始~结束|开始~结束
//	AtTime 仅 Once 时，绝对触发时刻
//	Span   相对时间窗（秒），嵌套在 With 段内；10~ 或 10 表示到 With 段结束
//	WaitRun false/不写=立即 Run；true=先按 Every 倒计时
//	Run    到点时执行的函数
type TimerValid struct {
	Every  int
	Before int
	After  int
	AtTime string
	Once   bool
	WaitRun bool
	With   string
	Span   string
	Run    func()
}

type timerTask struct {
	Label string
	TimerValid

	hubStart time.Time
	nextAt   time.Time
	running  bool
	waitRun  bool
	slot     int
	atSlots  [][]string
	spanFrom int
	spanTo   int
}

func (h *WsyTimer) Set(label string, v TimerValid) *WsyTimer {
	if v.Run != nil {
		h.pending = append(h.pending, timerTask{
			Label:      label,
			TimerValid: v,
		})
	}
	return h
}



func (h *WsyTimer) New() {
	all := h.pending
	h.pending = nil
	h.run(all)
}


func (h *WsyTimer) run(tasks []timerTask) {
	startAt := time.Now()
	list := make([]*timerTask, 0, len(tasks))
	for i := range tasks {
		if tasks[i].Run == nil {
			continue
		}
		t := tasks[i]
		t.hubStart = startAt
		t.slot = -1
		t.parseWith()
		t.parseSpan()
		t.initNext()
		list = append(list, &t)
	}
	h.mu.Lock()
	h.tasks = list
	h.mu.Unlock()
	for {
		now := time.Now()
		h.refreshShow(list, now)
		for _, t := range list {
			t.tick(now)
		}
		time.Sleep(time.Second)
	}
}

// SetEvery 运行中修改指定任务的 Every（秒），并重排当前倒计时；找到返回 true
func (h *WsyTimer) SetEvery(label string, every int) bool {
	if every < 0 {
		every = 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, t := range h.tasks {
		if t.Label == label {
			t.setEvery(time.Now(), every)
			return true
		}
	}
	return false
}

// Every 查询运行中任务的 Every（秒）；未找到返回 -1
func (h *WsyTimer) Every(label string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, t := range h.tasks {
		if t.Label == label {
			return t.Every
		}
	}
	return -1
}
func (h *WsyTimer) refreshShow(list []*timerTask, now time.Time) {
	parts := make([]string, 0, len(list))
	for _, t := range list {
		if line := t.statusLine(now); line != "" {
			parts = append(parts, line)
		}
	}
	line := Date.ToTime()
	if len(parts) > 0 {
		line += "  " + strings.Join(parts, " | ")
	}
	fmt.Printf("\r[%s]\033[K", line)
}

func (t *timerTask) secDur(sec int) time.Duration {
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

func (t *timerTask) durClamp(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func (t *timerTask) parseWith() {
	s := strings.TrimSpace(t.With)
	if s == "" {
		t.atSlots = nil
		return
	}
	var slots [][]string
	for _, part := range strings.Split(s, "|") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		seg := strings.SplitN(part, "~", 2)
		if len(seg) != 2 {
			continue
		}
		from, to := strings.TrimSpace(seg[0]), strings.TrimSpace(seg[1])
		if from != "" && to != "" {
			slots = append(slots, []string{from, to})
		}
	}
	t.atSlots = slots
}

func (t *timerTask) parseSpan() {
	s := strings.TrimSpace(t.Span)
	if s == "" {
		t.spanFrom, t.spanTo = 0, 0
		return
	}
	if !strings.Contains(s, "~") {
		t.spanFrom, t.spanTo = Str.ToInt(s), 0
		return
	}
	seg := strings.SplitN(s, "~", 2)
	t.spanFrom = Str.ToInt(strings.TrimSpace(seg[0]))
	if len(seg) == 2 && strings.TrimSpace(seg[1]) != "" {
		t.spanTo = Str.ToInt(strings.TrimSpace(seg[1]))
	}
}

func (t *timerTask) hasWindow() bool {
	return len(t.atSlots) > 0 || t.spanFrom > 0 || t.spanTo > 0
}

func (t *timerTask) currentAt(now time.Time) int {
	for i, s := range t.atSlots {
		if len(s) < 2 {
			continue
		}
		b, e := Date.ToStdTime(s[0]), Date.ToStdTime(s[1])
		if !now.Before(b) && now.Before(e) {
			return i
		}
	}
	return -1
}

func (t *timerTask) hasFutureAt(now time.Time) bool {
	for _, s := range t.atSlots {
		if len(s) > 0 && now.Before(Date.ToStdTime(s[0])) {
			return true
		}
	}
	return false
}

func (t *timerTask) slotBegin(i int) time.Time {
	if i < 0 || i >= len(t.atSlots) {
		return time.Time{}
	}
	anchor := Date.ToStdTime(t.atSlots[i][0])
	if t.spanFrom > 0 {
		return anchor.Add(t.secDur(t.spanFrom))
	}
	return anchor
}

func (t *timerTask) slotEnd(i int) time.Time {
	if i < 0 || i >= len(t.atSlots) || len(t.atSlots[i]) < 2 {
		return time.Time{}
	}
	anchor := Date.ToStdTime(t.atSlots[i][0])
	end := Date.ToStdTime(t.atSlots[i][1])
	if t.spanTo > 0 {
		if spanEnd := anchor.Add(t.secDur(t.spanTo)); spanEnd.Before(end) {
			end = spanEnd
		}
	}
	return end
}

func (t *timerTask) beginAt(now time.Time) time.Time {
	if len(t.atSlots) > 0 {
		if idx := t.currentAt(now); idx >= 0 {
			return t.slotBegin(idx)
		}
		var next time.Time
		nextIdx := -1
		for i, s := range t.atSlots {
			if len(s) < 1 {
				continue
			}
			b := Date.ToStdTime(s[0])
			if now.Before(b) && (next.IsZero() || b.Before(next)) {
				next, nextIdx = b, i
			}
		}
		if nextIdx >= 0 {
			return t.slotBegin(nextIdx)
		}
	}
	if t.spanFrom > 0 {
		return t.hubStart.Add(t.secDur(t.spanFrom))
	}
	return t.hubStart
}

func (t *timerTask) spanState(now time.Time) string {
	if t.Once && t.Run == nil {
		return "done"
	}
	if !t.hasWindow() {
		return "active"
	}
	if len(t.atSlots) > 0 {
		idx := t.currentAt(now)
		if idx < 0 {
			if !t.hasFutureAt(now) {
				return "end"
			}
			return "wait"
		}
		if now.Before(t.slotBegin(idx)) {
			return "wait"
		}
		if end := t.slotEnd(idx); !end.IsZero() && !now.Before(end) {
			return "spanEnd"
		}
		return "active"
	}
	begin := t.hubStart
	if t.spanFrom > 0 {
		begin = t.hubStart.Add(t.secDur(t.spanFrom))
	}
	if now.Before(begin) {
		return "wait"
	}
	if t.spanTo > 0 && !now.Before(t.hubStart.Add(t.secDur(t.spanTo))) {
		return "end"
	}
	return "active"
}

func (t *timerTask) fmtLeft(until, now time.Time) string {
	return t.formatCountdown(t.durClamp(until.Sub(now)))
}

func (t *timerTask) formatCountdown(d time.Duration) string {
	sec := int(d.Round(time.Second).Seconds())
	if sec >= 3600 {
		return fmt.Sprintf("%dh%02dm%02ds", sec/3600, (sec%3600)/60, sec%60)
	}
	if sec >= 60 {
		return fmt.Sprintf("%dm%02ds", sec/60, sec%60)
	}
	return fmt.Sprintf("%ds", sec)
}

func (t *timerTask) runNow(now time.Time) bool {
	if t.WaitRun {
		return false
	}
	return t.spanState(now) == "active"
}

func (t *timerTask) setEvery(now time.Time, every int) {
	t.Every = every
	if t.running || t.Run == nil || t.Once {
		return
	}
	if t.spanState(now) != "active" {
		return
	}
	if t.waitRun {
		return
	}
	delay := t.secDur(every)
	if delay <= 0 {
		delay = time.Second
	}
	t.nextAt = now.Add(delay)
}

func (t *timerTask) scheduleNext(now, anchor time.Time) {
	delay := t.secDur(t.Every)
	if delay <= 0 {
		delay = time.Second
	}
	if t.runNow(now) {
		t.nextAt = now
		return
	}
	if anchor.Before(now) {
		anchor = now
	}
	t.nextAt = anchor.Add(delay)
}

func (t *timerTask) initNext() {
	now := time.Now()
	base := t.beginAt(now)
	t.waitRun = false
	if t.Once {
		fireAt := Date.ToStdTime(t.AtTime)
		if t.AtTime == "" {
			fireAt = t.hubStart.Add(t.secDur(t.Every))
		}
		if fireAt.Before(base) {
			fireAt = base
		}
		if t.runNow(now) {
			t.nextAt = now
		} else {
			t.nextAt = fireAt
		}
		if len(t.atSlots) > 0 {
			ok := false
			for i := range t.atSlots {
				if !fireAt.Before(t.slotBegin(i)) && fireAt.Before(t.slotEnd(i)) {
					ok = true
					break
				}
			}
			if !ok {
				t.Run = nil
			}
		} else if t.spanTo > 0 && !t.nextAt.Before(t.hubStart.Add(t.secDur(t.spanTo))) {
			t.Run = nil
		}
		return
	}
	t.scheduleNext(now, base)
}

func (t *timerTask) syncAtSlot(now time.Time) {
	if len(t.atSlots) == 0 {
		return
	}
	idx := t.currentAt(now)
	if idx < 0 {
		t.slot = -1
		return
	}
	if idx != t.slot {
		t.slot = idx
		t.initNext()
	}
}

func (t *timerTask) statusLine(now time.Time) string {
	switch st := t.spanState(now); st {
	case "done", "end":
		return ""
	case "spanEnd":
		return t.Label + ":段内结束"
	case "wait":
		return t.Label + ":待启(" + t.fmtLeft(t.beginAt(now), now) + ")"
	}
	if t.running {
		return t.Label + ":执行中"
	}
	if t.waitRun {
		return t.Label + ":待执行(" + t.fmtLeft(t.nextAt, now) + ")"
	}
	return t.Label + ":" + t.fmtLeft(t.nextAt, now)
}

func (t *timerTask) tick(now time.Time) {
	if t.Run == nil || t.running {
		return
	}
	t.syncAtSlot(now)
	switch t.spanState(now) {
	case "done", "end", "wait", "spanEnd":
		return
	}
	if now.Before(t.nextAt) {
		return
	}
	if !t.waitRun && t.Before > 0 {
		t.waitRun = true
		t.nextAt = now.Add(t.secDur(t.Before))
		return
	}
	t.waitRun = false
	t.running = true
	t.Run()
	doneAt := time.Now()
	t.running = false
	if t.Once {
		t.Run = nil
		return
	}
	delay := t.secDur(t.After) + t.secDur(t.Every)
	if delay <= 0 {
		delay = time.Second
	}
	t.nextAt = doneAt.Add(delay)
	if idx := t.currentAt(doneAt); idx >= 0 {
		if end := t.slotEnd(idx); !end.IsZero() && !t.nextAt.Before(end) {
			if len(t.atSlots) > 0 && t.hasFutureAt(doneAt) {
				return
			}
			t.Run = nil
		}
	}
}
