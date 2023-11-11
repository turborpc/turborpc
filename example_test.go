package turborpc_test

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/turborpc/turborpc"
)

type Counter atomic.Int64

func (c *Counter) Add(ctx context.Context, delta int64) (int64, error) {
	return (*atomic.Int64)(c).Add(delta), nil
}

func Example_counter() {
	rpc := turborpc.NewServer(turborpc.WithServerJavaScriptClient())

	_ = rpc.Register(&Counter{})

	http.Handle("/rpc", rpc)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
  <title>Counter</title>
  <script src="/rpc"></script>
</head>
<body>
  <strong id="count"></strong>
  <button id="plus">+</button>
  <button id="minus">-</button>
  <script>
    const rpc = new Counter("/rpc");

    const setCount = (v) => document.getElementById("count").innerText = v;
    document.getElementById("plus").onclick = () => rpc.add(1).then(setCount);
    document.getElementById("minus").onclick = () => rpc.add(-1).then(setCount);

    rpc.add(0).then(setCount);
  </script>
</body>
		`))
	})

	http.ListenAndServe(":3000", nil)
}
