package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)


type Task struct {
	TaskID int
	Filenames []string
	Status string
	Type string
	Worker string
	AssignedAt time.Time
}

type CoordinatorState string

const (
    Mapping      CoordinatorState = "mapping"
    Reducing     CoordinatorState = "reducing"
    Completed    CoordinatorState = "completed"
)

type Coordinator struct {
	Tasks []Task
	NReduce int
	mutex sync.Mutex
	State CoordinatorState
	IntermediateFiles map[int][]string
}

func (c *Coordinator) mapTasksComplete() bool {
	for _, task := range c.Tasks {
		if task.Type == "MapTask" && task.Status != "completed" {
				return false
		}
	}
	return true
}

func (c *Coordinator) reduceTasksComplete() bool {
	for _, task := range c.Tasks {
			if task.Type == "ReduceTask" && task.Status != "completed" {
					return false
			}
	}
	return true
}

func (c *Coordinator) prepareReduceTasks() {
	// Prepare the reduce tasks in the Task list
	rTasks := []Task{}
	for i:=0; i < c.NReduce; i++ {
		if filenames, ok := c.IntermediateFiles[i]; ok {
			// Create a new Task for each reduce operation
			reduceTask := Task{
					TaskID:    i, 
					Filenames: filenames,
					Status:    "idle",
					Type:      "ReduceTask",
					Worker:    "", 
			}

			rTasks = append(rTasks, reduceTask)
			c.Tasks = rTasks
		}
		
	}
}

func (c *Coordinator) checkAndTransitionState() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	switch c.State {
	case Mapping:
		if c.mapTasksComplete() {
			c.State = Reducing
			c.prepareReduceTasks()
		}
	case Reducing:
		if c.reduceTasksComplete() {
			c.State = Completed
		}
	}
}



func (c *Coordinator) TaskAssign(args *TaskRequestArgs, reply *TaskReturn) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.State == Completed {
		reply.TaskType = "DoneTask"
		return nil
	}

	for i := range c.Tasks {
		task := &c.Tasks[i]
		taskEligible := task.Status == "idle" || (task.Status == "in-progress" && time.Since(task.AssignedAt) > 10*time.Second)
		if taskEligible {
			assignTask(task, args.WorkerID, reply)
			reply.NReduce = c.NReduce
			return nil
		}
	}
	return nil
}

func (c *Coordinator) MapTaskCompleted(args *MapTaskDoneArgs, reply *MapTaskDoneReply) error {
	c.mutex.Lock()
	c.Tasks[args.TaskId].Status = "completed"

	for _, filename := range args.IntermediateFiles {
		reduceTaskID := extractReduceTaskIDFromFilename(filename)
		c.IntermediateFiles[reduceTaskID] = append(c.IntermediateFiles[reduceTaskID], filename)
	}
	c.mutex.Unlock()
	c.checkAndTransitionState()
	return nil
}

func (c *Coordinator) ReduceTaskCompleted(args *ReduceTaskDoneArgs, reply *ReduceTaskDoneReply) error {
	c.mutex.Lock()
	c.Tasks[args.TaskId].Status = "completed"
	c.mutex.Unlock()
	c.checkAndTransitionState()
	return nil
}


//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	ret := false


	if c.State == Completed {
		ret = true
	}


	return ret
}


func MakeCoordinator(files []string, nReduce int) *Coordinator {
	
	tasks := taskInitaliser(files)
	c := Coordinator{
		Tasks: tasks, 
		State: Mapping, 
		NReduce: nReduce,
		IntermediateFiles: make(map[int][]string),
	}

	// Your code here.


	c.server()
	return &c
}

func assignTask(task *Task, workerID string, reply *TaskReturn) {
	task.Status = "in-progress"
	task.Worker = workerID
	task.AssignedAt = time.Now()

	reply.TaskId = task.TaskID
	reply.Filenames = task.Filenames
	reply.TaskType = task.Type
}

func taskInitaliser(files []string) []Task {
	res := make([]Task, len(files))
	for i:=0; i < len(files); i++ {
		res[i] = Task{
			TaskID: i,
			Filenames: []string{files[i]},
			Status: "idle",
			Type: "MapTask",
		}
	}
	return res
}

func extractReduceTaskIDFromFilename(filename string) int {
	parts := strings.Split(filename, "-")
	if len(parts) < 3 {
		fmt.Println("Invalid Filename")
		return -1
	}

	reduceTaskIDStr := parts[len(parts)-2]
	reduceTaskID, err := strconv.Atoi(reduceTaskIDStr)
	if err != nil {
		fmt.Println("Error converting to int")
		return -1
	}
	return reduceTaskID
}