package lib

import (
	"github.com/streadway/amqp"
	"os/exec"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"archive/zip"
	"path/filepath"
	"strings"
	"io"
	"sync"
	_"github.com/go-sql-driver/mysql"
	"io/ioutil"
	"fmt"
	"crypto/md5"
	"encoding/hex"
)

const StateFile = "state"

type Task struct {
	Tasks map[int64]*task
	Config common
	Database Database
	Common Common
	sync.RWMutex
}

/**
1 => [ // taskId
	InJob => 4 // tasks in job
	Completed => 3 // tasks is completed
	LastIsComplete => true // lastJob is complete
]
 */
type task struct {
	InJob int
	Completed int
	LastIsComplete bool
	CommonCount int
	ErrorCount int
	CompanyArchivePath string
}


type JsonTask struct {
	TaskId *int64 `json:",omitempty"`
	LastDoc *bool `json:",omitempty"`
	DocCount *int `json:",omitempty"`
	CompanyArchivePath *string
}

// running goroutine and return answer to rabbit
func (t * Task) RunJob(delivery amqp.Delivery) error {
	t.DoJob(delivery.Body)
	t.saveState()
	return delivery.Ack(false)
}

// parse json and set task to job
// running php script for doing job
// unset task from job
func (t * Task) DoJob(msgs []byte) {
	t.Lock()
	jsonTask := t.StartTask(msgs)
	t.Unlock()

	cmd := exec.Command(t.Config.PathToPhp, t.Config.PathToCreator, string(msgs[:]))
	stdoutErr, err := cmd.CombinedOutput()
	genDocErr := 0

	t.Common.FailOnError(err, "Error in exec command")
	if stdoutErr != nil && len(stdoutErr) != 0 {
		if string(stdoutErr[:]) == t.Config.DocGenError {
			genDocErr = 1
		}
		t.Common.WriteLog(t.Config.PathToPhpLog, stdoutErr)
	}

	t.Lock()

	t.FinishTask(jsonTask, genDocErr)
	t.FinishJob(jsonTask)

	t.Unlock()
}

// json_decode
// set values to Tasks
func (t *Task) StartTask(msg []byte) JsonTask {
	var jT JsonTask
	err := json.Unmarshal(msg, &jT)
	t.Common.FailOnError(err, "Error in json")

	if jT.LastDoc == nil || jT.TaskId == nil || jT.DocCount == nil {
		t.Common.FailOnError(errors.New("required fields in json is absent"), "Error in json")
	}

	if parentTask, ok := t.Tasks[*jT.TaskId]; ok {
		parentTask.InJob++
	} else {
		if len(t.Tasks) == 0 {
			t.Tasks = make(map[int64]*task)
		}

		path := ""
		if jT.CompanyArchivePath != nil {
			path = *jT.CompanyArchivePath
		}

		t.Tasks[*jT.TaskId] = &task{
			InJob : 1,
			Completed: 0,
			LastIsComplete: false,
			CommonCount: *jT.DocCount,
			ErrorCount: 0,
			CompanyArchivePath: path,
		}
	}

	return jT
}

func (t *Task) FinishTask(jt JsonTask, errCount int) {
	t.Tasks[*jt.TaskId].Completed++
	t.Tasks[*jt.TaskId].InJob--
	t.Tasks[*jt.TaskId].ErrorCount += errCount
	if *jt.LastDoc {
		t.Tasks[*jt.TaskId].LastIsComplete = true
	}
}

// if all completed
// no tasks in job
// lastDoc is complete
func (t *Task) FinishJob(jt JsonTask) {

	ttask := t.Tasks[*jt.TaskId]

	if ttask.InJob == 0 && ttask.Completed != 0 && ttask.Completed == ttask.CommonCount && ttask.LastIsComplete == true {
		arcError := t.ToArchive(*jt.TaskId)
		t.ToDatabase(*jt.TaskId, true)
		t.RunPostScript(*jt.TaskId, arcError)
		delete(t.Tasks, *jt.TaskId)
	}
}

func (t *Task) RunPostScript(taskId int64, arcError error) {
	archiveError := 0
	if arcError != nil {
		archiveError = 1
	}
	cmd := exec.Command(
		t.Config.PathToPhp,
		t.Config.PathToPostScript,
		strconv.Itoa(int(taskId)),
		strconv.Itoa(archiveError),
		strconv.Itoa(t.Tasks[taskId].ErrorCount),
	)
	_, err := cmd.CombinedOutput()
	t.Common.FailOnError(err, "Error in exec command")
}

func (t *Task) ToArchive(taskId int64) error {
	taskDir := t.Config.PathToResultDoc + "/" + strconv.FormatInt(taskId, 10)
	info, err := os.Stat(taskDir)
	t.Common.FailOnError(err, "Check dir of task")

	if !info.IsDir() {
		dirError := "dir of task is not dir"
		t.Common.FailOnError(err, dirError)
		return errors.New(dirError)
	}


	baseDir := filepath.Base(taskDir)

	// create dir to archive and zip file
	companySubDir := t.Tasks[taskId].CompanyArchivePath + "/"
	pathToArchive := t.Config.PathToResultZip + companySubDir
	info, err = os.Stat(pathToArchive)
	if os.IsNotExist(err) {
		err := os.MkdirAll(pathToArchive, 0775)
		t.Common.FailOnError(err, "Error in creating archive Dir")
	}

	zipfile, err := os.Create( pathToArchive + strconv.FormatInt(taskId, 10) + ".zip")
	t.Common.FailOnError(err, "creating zip file")
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	arcErr := filepath.Walk(taskDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, taskDir))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)

		return err
	})

	t.Common.FailOnError(arcErr, "Error in putting file to archive")

	err = os.RemoveAll(taskDir)
	t.Common.FailOnError(err, "Deleting files")

	return arcErr
}

func (t *Task) ToDatabase (taskId int64, result bool) {
	db := t.Database.Open(t.Common)
	defer db.Close()

	updateTask, err := db.Prepare(
		"UPDATE " + t.Database.Config.Table + " SET " +
		t.Database.Config.ResultField + " = ?, " + t.Database.Config.LogField +
		" = ?, " + t.Database.Config.CodeField + " = ? WHERE " + t.Database.Config.TaskIdField + "= ?")

	t.Common.FailOnError(err, "Prepare update statement error")
	defer updateTask.Close()

	resultCode := t.Database.Config.SuccessCode
	if result == false {
		resultCode = t.Database.Config.ErrorCode
	}

	taskCode := md5.Sum([]byte(strconv.Itoa(int(taskId)) + t.Config.DocCodeSalt))
	errorJson := "[{\"errorsCount\": " + strconv.Itoa(t.Tasks[taskId].ErrorCount) + "}]"

	_, err = updateTask.Exec(resultCode, errorJson, hex.EncodeToString(taskCode[:]), taskId)
	t.Common.FailOnError(err, "Error in execute update statement")
}

func (t *Task) saveState() {
	data, err := json.Marshal(t.Tasks)
	t.Common.FailOnError(err, "Error in create json from task object")

	err = ioutil.WriteFile(StateFile, []byte(data), 0750)
	t.Common.FailOnError(err, "Error in writing state file")
}

// if script has been stoped, but not all tasks has been evaluated
// demon can recover their state from state file
func (t *Task) RecoverState() {
	if _, err := os.Stat(StateFile); os.IsNotExist(err) {
		return
	}

	data, err := ioutil.ReadFile(StateFile)
	t.Common.FailOnError(err, "Error in reading state file")
	if err != nil {
		return
	}

	validJson := json.Valid(data)
	if validJson == false {
		t.Common.WriteLog(
			t.Config.PathToGoLog,
			[]byte(fmt.Sprintf("%s: %s", "Recovering data from state file", "no valid data in file")))
		return
	}

	err = json.Unmarshal(data, &t.Tasks)
	t.Common.FailOnError(err, "Recovering data from state file")
	if err != nil {
		return
	}

	for _, v := range t.Tasks {
		v.InJob = 0
	}
}
