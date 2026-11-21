# Testing DataLoader - Step by Step Guide

## ✅ Setup Complete!

I've added a new GraphQL query `JobsByIds` that uses DataLoader. This will trigger your `JobBatchFunc` and show the logs.

## 🚀 How to Test

### Step 1: Start the Application

```bash
./job-queue
```

Or if you want to run directly:

```bash
go run main.go
```

### Step 2: Open GraphiQL

Navigate to: http://localhost:58579/graphiql

### Step 3: Create Some Jobs First

Run this mutation a few times to create jobs:

```graphql
mutation {
  job1: Enqueue(task: "task-1") {
    id
    task
    status
  }
}
```

```graphql
mutation {
  job2: Enqueue(task: "task-2") {
    id
    task
    status
  }
}
```

```graphql
mutation {
  job3: Enqueue(task: "task-3") {
    id
    task
    status
  }
}
```

**Copy the job IDs from the responses!**

### Step 4: Test DataLoader with JobsByIds

Now use the job IDs you copied to test the DataLoader:

```graphql
query {
  JobsByIds(ids: ["<job-id-1>", "<job-id-2>", "<job-id-3>"]) {
    id
    task
    status
    attempts
  }
}
```

**Replace `<job-id-1>`, `<job-id-2>`, `<job-id-3>` with actual job IDs!**

### Step 5: Check the Logs

In your terminal where the app is running, you should see:

```
INFO    JobBatchFunc    {"ids": ["job-id-1", "job-id-2", "job-id-3"]}
```

🎉 **This proves DataLoader is working and batching the requests!**

## 📊 Alternative Test: Get All Jobs First

If you want to get job IDs easily:

```graphql
query {
  Jobs {
    id
    task
    status
  }
}
```

Copy 2-3 job IDs from the response, then use them in the `JobsByIds` query.

## 🔍 What's Happening?

1. **Without DataLoader** (old `Jobs` query):

   - Calls `jobService.GetAllJobs()` directly
   - No batching, no DataLoader involvement
   - ❌ No `JobBatchFunc` log

2. **With DataLoader** (new `JobsByIds` query):
   - Calls `loader.Load(ctx, key)` for each ID
   - DataLoader batches all requests
   - Calls `JobBatchFunc` once with all IDs
   - ✅ You see the log!

## 🧪 Advanced Test: Multiple Queries in One Request

Try this to see DataLoader batching multiple queries:

```graphql
query {
  job1: JobsByIds(ids: ["id1", "id2"]) {
    id
    task
  }
  job2: JobsByIds(ids: ["id3", "id4"]) {
    id
    task
  }
}
```

DataLoader will batch ALL 4 IDs into a single `JobBatchFunc` call!

## 🐛 Troubleshooting

### No logs appearing?

1. **Check if the app is running**: Make sure `./job-queue` is running
2. **Check job IDs are correct**: Use real job IDs from created jobs
3. **Check log level**: The app uses `zap.L().Info()` which should show by default
4. **Rebuild after changes**: Run `make bind-static && go build -o job-queue main.go`

### Error: "Unknown field JobsByIds"

- Run `make bind-static` to regenerate the schema
- Rebuild: `go build -o job-queue main.go`
- Restart the application

## 📝 Example Complete Flow

```bash
# 1. Start app
./job-queue

# 2. In browser, go to http://localhost:58579/graphiql

# 3. Create jobs
mutation { Enqueue(task: "test-1") { id } }
mutation { Enqueue(task: "test-2") { id } }
mutation { Enqueue(task: "test-3") { id } }

# 4. Get all jobs to see IDs
query { Jobs { id } }

# 5. Test DataLoader (use real IDs)
query {
  JobsByIds(ids: ["550e8400-...", "6ba7b810-...", "7c9e6679-..."]) {
    id
    task
    status
  }
}

# 6. Check terminal - you should see:
# INFO    JobBatchFunc    {"ids": [...]}
```

## ✨ Success Criteria

You know DataLoader is working when:

- ✅ You see `JobBatchFunc` log in the terminal
- ✅ The log shows all requested IDs in a single array
- ✅ The query returns the correct job data

Happy testing! 🚀
