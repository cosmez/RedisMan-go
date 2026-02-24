package conn

import (
	"context"
	"errors"
	"iter"
	"net"
	"time"

	"github.com/cosmez/redisman-go/internal/resp"
)

// Subscribe listens for messages on a subscribed channel until the context is cancelled.
//
// C#:
// public IEnumerable<IRedisValue> Subscribe(CancellationToken token)
//
// Go:
// We use context.Context for cancellation. We set a short read deadline in a loop
// so we can periodically check ctx.Err() without blocking forever on Receive.
func (c *Connection) Subscribe(ctx context.Context) iter.Seq[resp.RedisValue] {
	return func(yield func(resp.RedisValue) bool) {
		for {
			// Check if context is cancelled before attempting to read
			if err := ctx.Err(); err != nil {
				return
			}

			// Use a short timeout so we can check ctx.Err() frequently
			response, err := c.Receive(200 * time.Millisecond)

			if err != nil {
				// If it's a timeout, just loop and check context again
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					continue
				}

				// For other errors, yield them and stop
				yield(resp.RedisError{Value: err.Error()})
				return
			}

			// Yield the received message
			if !yield(response) {
				return // Consumer stopped iterating
			}
		}
	}
}
