package ncsoft

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// Snapshot is everything the official site exposes about one character,
// fetched in one go. It is the input to the stat rebuild and the engine.
type Snapshot struct {
	Ref
	FetchedAt time.Time                     `json:"fetchedAt"`
	Info      *InfoResponse                 `json:"info"`
	Equipment *EquipmentResponse            `json:"equipment"`
	Items     map[string]*ItemResponse      `json:"items"`     // keyed by slotPos
	Daevanion map[string]*DaevanionResponse `json:"daevanion"` // keyed by board id
}

// LoadSnapshot fetches profile, equipment, every equipped item and every open
// Daevanion board. A full load is roughly 30 upstream requests.
func (c *Client) LoadSnapshot(ctx context.Context, ref Ref) (*Snapshot, error) {
	info, err := c.Info(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("info: %w", err)
	}
	eq, err := c.Equipment(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("equipment: %w", err)
	}
	s := &Snapshot{
		Ref:       ref,
		FetchedAt: time.Now().UTC(),
		Info:      info,
		Equipment: eq,
		Items:     map[string]*ItemResponse{},
		Daevanion: map[string]*DaevanionResponse{},
	}
	// Items and boards are fetched concurrently; the client still spaces
	// request starts by MinGap, so this only overlaps upstream latency.
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
	)
	fail := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}
	for _, slot := range eq.Equipment.EquipmentList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			it, err := c.Item(ctx, ref, slot)
			if err != nil {
				fail(fmt.Errorf("item slot %d: %w", slot.SlotPos, err))
				return
			}
			mu.Lock()
			s.Items[strconv.Itoa(slot.SlotPos)] = it
			mu.Unlock()
		}()
	}
	for _, b := range info.Daevanion.BoardList {
		if b.Open == 0 || b.OpenNodeCount == 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := c.Daevanion(ctx, ref, b.ID)
			if errors.Is(err, ErrNotFound) {
				return
			}
			if err != nil {
				fail(fmt.Errorf("daevanion board %d: %w", b.ID, err))
				return
			}
			mu.Lock()
			s.Daevanion[strconv.Itoa(b.ID)] = d
			mu.Unlock()
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return s, nil
}
