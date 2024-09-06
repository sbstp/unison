package unison

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestSidekickGroup_Basics(t *testing.T) {
	testCases := []struct {
		name   string
		result string
		err    error
		panic  bool
	}{
		{
			name:   "ok",
			result: "hello",
			err:    nil,
		},
		{
			name:   "error",
			result: "no",
			err:    fmt.Errorf("error!"),
		},
		{
			name:  "panic",
			panic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewSidekickGroup[string](context.Background())
			g.Main(func(ctx context.Context) (string, error) {
				if tc.panic {
					panic("boom")
				}
				return tc.result, tc.err
			})
			if !tc.panic {
				res, err := g.Wait()
				if err != tc.err {
					t.Fatalf("invalid err: got %v expected %v", err, tc.err)
				}
				if res != tc.result {
					t.Fatalf("invalid result: got %q expected %q", res, tc.result)
				}
			} else {
				defer func() {
					recover()
				}()
				g.Wait()
				t.Fatalf("did not panic")
			}

		})
	}
}

func TestSidekickGroup_MainCancel(t *testing.T) {
	var mainErr error

	g := NewSidekickGroup[string](context.Background())
	g.Main(func(ctx context.Context) (string, error) {
		<-ctx.Done()
		mainErr = ctx.Err()
		return "", ctx.Err()
	})
	g.Sidekick(func(ctx context.Context) error {
		return fmt.Errorf("error!")
	})
	res, err := g.Wait()
	if res != "" {
		t.Fatal("invalid res")
	}
	if err.Error() != "error!" {
		t.Fatal("invalid err")
	}
	if !errors.Is(mainErr, context.Canceled) {
		t.Fatal("not canceled")
	}
}

func TestSidekickGroup_SidekickCancel(t *testing.T) {
	var sidekickErr error

	g := NewSidekickGroup[string](context.Background())
	g.Main(func(ctx context.Context) (string, error) {
		return "hello", nil
	})
	g.Sidekick(func(ctx context.Context) error {
		<-ctx.Done()
		sidekickErr = ctx.Err()
		return ctx.Err()
	})
	res, err := g.Wait()
	if res != "hello" {
		t.Fatal("invalid res")
	}
	if err != nil {
		t.Fatal("invalid err")
	}
	if !errors.Is(sidekickErr, context.Canceled) {
		t.Fatal("not canceled")
	}
}
