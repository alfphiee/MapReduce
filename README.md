# Map Reduce - Lab 1


I'm currently watching the MIT 6.824 Distributed Systems lectures - with which the Labs are available [here](https://pdos.csail.mit.edu/6.824/schedule.html) with Lab 1 being available [here](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)


The goal of this lab was to implement a MapReduce system - by implementing:
- A worker process that:
  - Calls application Map and Reduce functions
  - Handles reading and writing
- A coordinator process that:
  - Hands out tasks to workers
  - Copes with failed Workers


## Get Started


To run this implementation:


1) Clone this repository
2) Navigate into the repository's directory with `cd src/main`
3) run `go build -buildmode=plugin ../mrapps/wc.go` to build the Word Count plugin
4) run `go run mrcoordinator.go pg*.txt` to create a Coordinator
5) run `go run mrworker.go wc.so` to run a worker (repeat in multiple terminals to run multiple workers)


To run tests:


1) `bash test-mr.sh` to run test suite once
2) `bash test-mr-many.sh X` - with X representing number of times to run tests


## Implementation Details


- **Workers Request Tasks**: Workers periodically request tasks from the coordinator, which can be either Map or Reduce tasks depending on the job's current phase.
- **Coordinator Assigns Tasks**: The coordinator keeps track of all tasks' states—whether they are idle, in progress, or completed. Based on this state, it assigns tasks to requesting workers.
- **Workers Execute and Report Completion**: Workers execute the assigned tasks. For Map tasks, they read input files, apply the Map function, and write the output to intermediate files. For Reduce tasks, they aggregate intermediate files and apply the Reduce function. Upon completion, workers report their status back to the coordinator.
- **Job Completion**: Once all tasks (Map and Reduce) have been completed, the coordinator marks the entire job as done. This state is communicated to workers to indicate that no further tasks are available.


Failure handling is a critical aspect of the system. If a worker fails to complete a task within a certain timeframe, the coordinator reassigns the task, considering it as idle again. This ensures robustness against worker failures.




## Design Choices


### Task Management


I initially implemented a unified list to manage both Map and Reduce tasks. This approach simplified the initial design but introduced complexity when transitioning between the mapping and reducing phases. Reflecting on this, a design separating Map and Reduce tasks into distinct lists might offer clearer state management and transition handling, albeit at the complexity cost of managing two lists.


### Task Availability and Assignment


The current implementation iterates through the task list to find available tasks, which, while straightforward, may not be the most efficient, especially as the number of tasks grows. A future enhancement could involve using a counter to track the number of idle tasks directly, allowing for quicker task assignments and reducing the need for iteration.


## Challenges Faced


Primary challenge I faced was that this was my first exposure to Go and also I don't have much experience with Distributed Systems


I was able to pick up Go without too much difficulty from the existing code that came with the Lab as well as by completing the 'Go Tour'.


Using Google I was able to find out potential ways to circumvent issues mentioned in the Lab such as 'Race Conditions' - I did this by using Go's `sync.Mutex` type to attempt to control concurrent access to shared resources.


Another challenge was actually ensuring I had a solid understanding of MapReduce as a process - I did this by reading the [paper](https://static.googleusercontent.com/media/research.google.com/en//archive/mapreduce-osdi04.pdf) and from watching the 6.824 lectures.


## Future Improvements


I'm sure there are problems with my implementation that I'm not aware of currently - I'd like to come back to this after I gain a better understanding in the area.


Additionally I would like to clean up my code in some place - due to lack of familiarity I feel that my code isn't always clear and concise.


I would also like to improve the logic around reassigning tasks from potentially faulty workers - currently when assigning tasks I check if the Task is idle or if it is in-progress for more than 10 seconds - if the task meets either of these conditions it is assigned. I think I prefer a solution that separates the detection of slow 'workers' from the assigning of tasks.




## Resources


- [Go Programming Language Documentation](https://golang.org/doc/)
- [MIT 6.824: Distributed Systems](https://pdos.csail.mit.edu/6.824/schedule.html)
- [Google's MapReduce Paper](https://static.googleusercontent.com/media/research.google.com/en//archive/mapreduce-osdi04.pdf)







