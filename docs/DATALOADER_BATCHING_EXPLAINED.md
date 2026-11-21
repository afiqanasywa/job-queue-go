# Why Batch Size Was 1 - DataLoader Batching Explained

## 🔍 The Problem

You saw this in your logs:

```
JobBatchFunc called - DataLoader batching!  {"job_ids": ["id-1"], "batch_size": 1}
JobBatchFunc called - DataLoader batching!  {"job_ids": ["id-2"], "batch_size": 1}
```

**Expected:** One batch with all IDs
**Actual:** Multiple batches with 1 ID each

## ❌ The Bug - Original Code

```go
func (q JobQuery) JobsByIds(ctx context.Context, args struct {
    IDs []string
}) ([]resolver.JobResolver, error) {
    loader := q.dataloader.JobLoader

    for _, id := range args.IDs {
        thunk := loader.Load(ctx, dataloader.StringKey(id))
        result, err := thunk()  // ❌ PROBLEM: Executing immediately!
        // Process result...
    }
}
```

### Why This Doesn't Batch

1. **Loop iteration 1:**

   - `loader.Load(id1)` - Queues id1
   - `thunk()` - **Executes immediately**, triggers `JobBatchFunc` with only `[id1]`

2. **Loop iteration 2:**
   - `loader.Load(id2)` - Queues id2
   - `thunk()` - **Executes immediately**, triggers `JobBatchFunc` with only `[id2]`

Each `thunk()` call forces DataLoader to execute before it can collect other requests!

## ✅ The Fix - Corrected Code

```go
func (q JobQuery) JobsByIds(ctx context.Context, args struct {
    IDs []string
}) ([]resolver.JobResolver, error) {
    loader := q.dataloader.JobLoader

    // STEP 1: Queue all Load() calls first
    thunks := make([]dataloader.Thunk, 0, len(args.IDs))
    for _, id := range args.IDs {
        thunk := loader.Load(ctx, dataloader.StringKey(id))
        thunks = append(thunks, thunk)  // ✅ Just collect, don't execute!
    }

    // STEP 2: Now execute all thunks together
    resolvers := make([]resolver.JobResolver, 0, len(args.IDs))
    for _, thunk := range thunks {
        result, err := thunk()  // ✅ Now DataLoader can batch!
        // Process result...
    }
}
```

### Why This Works

1. **First loop (STEP 1):**

   - `loader.Load(id1)` - Queues id1
   - `loader.Load(id2)` - Queues id2
   - `loader.Load(id3)` - Queues id3
   - **No execution yet!** All IDs are queued

2. **Second loop (STEP 2):**
   - First `thunk()` call triggers DataLoader
   - DataLoader sees 3 queued requests
   - Calls `JobBatchFunc` **once** with `[id1, id2, id3]`
   - Subsequent `thunk()` calls return cached results

## 🎯 How DataLoader Batching Works

### DataLoader's Internal Mechanism

```
┌─────────────────────────────────────────────────────┐
│  DataLoader Batching Timeline                       │
├─────────────────────────────────────────────────────┤
│                                                      │
│  Time 0ms:  Load(id1) ──┐                          │
│  Time 1ms:  Load(id2) ──┼─► Queue: [id1, id2, id3] │
│  Time 2ms:  Load(id3) ──┘                          │
│                                                      │
│  Time 3ms:  thunk() ──► Triggers batch execution   │
│                         ↓                           │
│                    JobBatchFunc([id1, id2, id3])   │
│                         ↓                           │
│                    Returns all results              │
│                                                      │
│  Time 4ms:  thunk() ──► Returns cached result      │
│  Time 5ms:  thunk() ──► Returns cached result      │
└─────────────────────────────────────────────────────┘
```

### Key Principles

1. **Queueing Phase**: `Load()` calls add keys to a queue
2. **Execution Phase**: First `thunk()` call triggers batch execution
3. **Caching**: Results are cached, subsequent `thunk()` calls are instant

## 📊 Comparison

### ❌ Without Proper Batching (Original)

```
Query: JobsByIds(ids: ["id1", "id2", "id3"])

Repository calls:
  1. FindByIDs(["id1"])     ← Batch size: 1
  2. FindByIDs(["id2"])     ← Batch size: 1
  3. FindByIDs(["id3"])     ← Batch size: 1

Total repository calls: 3
```

### ✅ With Proper Batching (Fixed)

```
Query: JobsByIds(ids: ["id1", "id2", "id3"])

Repository calls:
  1. FindByIDs(["id1", "id2", "id3"])  ← Batch size: 3

Total repository calls: 1  🎉
```

## 🧪 Testing the Fix

### Test Query

```graphql
query {
  JobsByIds(
    ids: [
      "86a1b70b-b9bf-47fd-b431-d0e0e3a14d0f"
      "0d6cdaf6-d452-4856-bd4a-8eda7080bf89"
      "550e8400-e29b-41d4-a716-446655440000"
    ]
  ) {
    id
    task
    status
  }
}
```

### Expected Log (After Fix)

```
INFO  JobBatchFunc called - DataLoader batching!  {
  "job_ids": [
    "86a1b70b-b9bf-47fd-b431-d0e0e3a14d0f",
    "0d6cdaf6-d452-4856-bd4a-8eda7080bf89",
    "550e8400-e29b-41d4-a716-446655440000"
  ],
  "batch_size": 3
}
```

**One log entry with all 3 IDs!** ✅

## 🎓 Key Takeaways

### Do's ✅

```go
// Collect all Load() calls first
thunks := []dataloader.Thunk{}
for _, id := range ids {
    thunk := loader.Load(ctx, dataloader.StringKey(id))
    thunks = append(thunks, thunk)  // Just collect
}

// Then execute
for _, thunk := range thunks {
    result, _ := thunk()  // Now batching works!
}
```

### Don'ts ❌

```go
// Don't execute thunk() immediately after Load()
for _, id := range ids {
    thunk := loader.Load(ctx, dataloader.StringKey(id))
    result, _ := thunk()  // ❌ Prevents batching!
}
```

## 🔬 Advanced: DataLoader Tick Mechanism

DataLoader batches requests within a single "tick" (event loop cycle). When you call `thunk()`:

1. **First `thunk()` call**: Triggers batch execution
2. **Same tick**: All queued requests are batched together
3. **Next tick**: New batch starts

By separating `Load()` from `thunk()`, you ensure all loads happen in the same tick!

## 💡 Real-World Example

### Scenario: Fetching User Posts

```go
// ❌ Bad: N queries
for _, userID := range userIDs {
    thunk := loader.Load(ctx, dataloader.StringKey(userID))
    user, _ := thunk()  // Executes immediately
    fmt.Println(user.Name)
}
// Result: 100 users = 100 database queries

// ✅ Good: 1 query
thunks := []dataloader.Thunk{}
for _, userID := range userIDs {
    thunks = append(thunks, loader.Load(ctx, dataloader.StringKey(userID)))
}
for _, thunk := range thunks {
    user, _ := thunk()  // First call batches all!
    fmt.Println(user.Name)
}
// Result: 100 users = 1 database query 🎉
```

## 🚀 Performance Impact

### Before Fix (No Batching)

- 10 jobs = 10 repository calls
- 100 jobs = 100 repository calls
- **O(n) complexity**

### After Fix (With Batching)

- 10 jobs = 1 repository call
- 100 jobs = 1 repository call
- **O(1) complexity** 🚀

## ✨ Summary

**The fix is simple but crucial:**

1. **Separate concerns**: Queue all loads, then execute
2. **Don't mix**: Don't call `thunk()` in the same loop as `Load()`
3. **Let DataLoader batch**: Give it time to collect all requests

Now your DataLoader will properly batch all requests into a single repository call! 🎉
