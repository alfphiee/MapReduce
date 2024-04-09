package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}


//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	pid := os.Getpid()
	workerID := strconv.Itoa(pid)

	toBreak := false
	for {
		task := RequestTask(workerID)
		switch task.TaskType {
		case "MapTask":
			intermediateFiles := applyMap(task, mapf)
			if len(intermediateFiles) == task.NReduce {
				NotifyMapTaskComplete(task.TaskId, intermediateFiles)
			}
		case "ReduceTask":
			applyReduce(task, reducef)
			NotifyReduceTaskComplete(task.TaskId)
		case "DoneTask":
			toBreak = true
		default:
			time.Sleep(2*time.Second)
		}
		if toBreak {
			break
		}

	}

}

func RequestTask(workerID string) TaskReturn{
	args := TaskRequestArgs{WorkerID: workerID}
	reply := TaskReturn{}
	call("Coordinator.TaskAssign", &args, &reply)
	return reply
}

func NotifyMapTaskComplete(taskId int, intermediateFiles []string) {
	args := MapTaskDoneArgs{
		TaskId: taskId,
		IntermediateFiles: intermediateFiles,
	}
	reply := MapTaskDoneReply{}

	call("Coordinator.MapTaskCompleted", &args, &reply)
}

func NotifyReduceTaskComplete(taskId int) {
	args := ReduceTaskDoneArgs{
		TaskId: taskId,
	}
	reply := ReduceTaskDoneReply{}

	call("Coordinator.ReduceTaskCompleted", &args, &reply)
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

func applyReduce(task TaskReturn, reducef func(string, []string) string) {
	intermediate := make(map[string][]string)
	for _, filename := range task.Filenames {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate[kv.Key] = append(intermediate[kv.Key], kv.Value)
		}
		file.Close()
	}

	var res []KeyValue
	for key, values := range intermediate {
		reducedValue := reducef(key, values)
		res = append(res, KeyValue{Key: key, Value: reducedValue})
	}

	outputFilename := "mr-out-" + strconv.Itoa(task.TaskId)
	file, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("cannot create %v", outputFilename)
	}
	defer file.Close()

	for _, kv := range res {
		_, err := fmt.Fprintf(file,"%v %v\n", kv.Key, kv.Value)
		if err != nil {
			log.Fatalf("cannot write to %v", outputFilename)
		}
	}

}

func applyMap(task TaskReturn, mapf func(string, string) []KeyValue) []string {
	filename := task.Filenames[0]
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	kva := mapf(filename, string(content))

	encoders := make([]*json.Encoder, task.NReduce)
	intermediateFiles := make([]string, task.NReduce)
	for i:=0; i < task.NReduce; i++ {
		tmpFile, err := os.CreateTemp(".", fmt.Sprintf("mr-%d-%d-", task.TaskId, i))
		if err != nil {
			log.Fatalf("cannot create temp file")
		}
		defer tmpFile.Close()
		intermediateFiles[i] = tmpFile.Name()
		encoders[i] = json.NewEncoder(tmpFile)
	}

	for _, kv := range kva {
		reduceTaskNumber := int(ihash(kv.Key)) % task.NReduce
		err := encoders[reduceTaskNumber].Encode(&kv)
		if err != nil {
			log.Fatalf("cannot write to temp file")
		}
	}
	return intermediateFiles
}