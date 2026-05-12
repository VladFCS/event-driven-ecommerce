package service

import "fmt"

type cancelIdempotencyRecord struct {
	fingerprint string
	done        chan struct{}
	result      *CancelOrderResult
	err         error
}

func cancelRequestFingerprint(orderID, reason string) string {
	return orderID + "\n" + reason
}

func (s *GatewayService) beginCancelIdempotency(key, fingerprint string) (*cancelIdempotencyRecord, bool, error) {
	s.cancelIdempotencyMu.Lock()
	defer s.cancelIdempotencyMu.Unlock()

	if record, ok := s.cancelIdempotency[key]; ok {
		if record.fingerprint != fingerprint {
			return nil, false, fmt.Errorf("%w: idempotency key reused with different cancel request", ErrIdempotencyConflict)
		}

		return record, true, nil
	}

	record := &cancelIdempotencyRecord{
		fingerprint: fingerprint,
		done:        make(chan struct{}),
	}
	s.cancelIdempotency[key] = record

	return record, false, nil
}

func (s *GatewayService) finishCancelIdempotency(key string, record *cancelIdempotencyRecord, result *CancelOrderResult, err error) {
	s.cancelIdempotencyMu.Lock()
	defer s.cancelIdempotencyMu.Unlock()

	if err != nil {
		record.err = err
		close(record.done)
		delete(s.cancelIdempotency, key)
		return
	}

	if result != nil {
		resultCopy := *result
		record.result = &resultCopy
	}

	close(record.done)
}
