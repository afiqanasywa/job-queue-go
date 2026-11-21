# GraphQL Playground Examples

## 1. SimultaneousCreateJob - Create Multiple Jobs Concurrently

This mutation creates three jobs at once to test concurrent job creation.

```graphql
mutation SimultaneousCreateJob {
  job1: Enqueue(task: "process-order-1") {
    id
    task
    status
    attempts
  }
  job2: Enqueue(task: "send-email-2") {
    id
    task
    status
    attempts
  }
  job3: Enqueue(task: "generate-report-3") {
    id
    task
    status
    attempts
  }
}
```

---

## 2. SimulateUnstableJob - Test Retry Logic

This creates a special "unstable-job" that fails twice before succeeding on the third attempt.

```graphql
mutation SimulateUnstableJob {
  Enqueue(task: "unstable-job") {
    id
    task
    status
    attempts
  }
}
```

**Note:** The job will be created with status "pending" and attempts 0. Wait a few seconds and query the job by ID to see it eventually complete with 3 attempts.

---

## 3. GetAllJobs - Retrieve All Jobs

Get a list of all jobs in the system.

```graphql
query GetAllJobs {
  Jobs {
    id
    task
    status
    attempts
  }
}
```

---

## 4. GetJobById - Retrieve Specific Job

Get details of a specific job by its ID. Replace `"your-job-id-here"` with an actual job ID from previous mutations.

```graphql
query GetJobById {
  Job(id: "your-job-id-here") {
    id
    task
    status
    attempts
  }
}
```

---

## 5. GetAllJobStatus - Get Job Statistics

Get statistics showing counts of jobs by status.

```graphql
query GetAllJobStatus {
  JobStatus {
    pending
    running
    failed
    completed
  }
}
```

---

## Complete Test Flow

Here's a recommended flow to test all functionality:

### Step 1: Create multiple jobs simultaneously
```graphql
mutation {
  job1: Enqueue(task: "task-1") { id task status }
  job2: Enqueue(task: "task-2") { id task status }
  job3: Enqueue(task: "task-3") { id task status }
}
```

### Step 2: Create an unstable job
# a0a39bf3-d29c-48a5-a6ec-1461cc7d4460

```graphql
mutation {
  unstableJob: Enqueue(task: "unstable-job") {
    id
    task
    status
    attempts
  }
}
```
**Save the ID returned for the next step!**

### Step 3: Wait 5-10 seconds, then check the unstable job status
```graphql
query {
  Job(id: "PASTE-UNSTABLE-JOB-ID-HERE") {
    id
    task
    status
    attempts
  }
}
```
You should see `attempts: 3` and `status: "completed"`

### Step 4: Get all jobs
```graphql
query {
  Jobs {
    id
    task
    status
    attempts
  }
}
```

### Step 5: Get job statistics
```graphql
query {
  JobStatus {
    pending
    running
    failed
    completed
  }
}
```

---

## Advanced: Create Many Jobs at Once

Test concurrent processing with many jobs:

```graphql
mutation CreateManyJobs {
  j1: Enqueue(task: "job-1") { id }
  j2: Enqueue(task: "job-2") { id }
  j3: Enqueue(task: "job-3") { id }
  j4: Enqueue(task: "job-4") { id }
  j5: Enqueue(task: "job-5") { id }
  j6: Enqueue(task: "unstable-job") { id }
  j7: Enqueue(task: "job-7") { id }
  j8: Enqueue(task: "job-8") { id }
  j9: Enqueue(task: "job-9") { id }
  j10: Enqueue(task: "job-10") { id }
}
```

Then immediately check the status:
```graphql
query {
  JobStatus {
    pending
    running
    failed
    completed
  }
}
```

Wait a few seconds and check again to see jobs complete!
