package lib

import (
//	"log"
	"os"
	"fmt"
	"log"
	"syscall"
)

type Common struct {
	Config common
}

func (c *Common) FailOnError(err error, msg string) {
	if err != nil {
		c.WriteLog(c.Config.PathToGoLog, []byte(fmt.Sprintf("%s: %s", msg, err)))
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func (c *Common) WriteLog(path string, msg []byte) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("%s: %s", "Error in opening log flie", err)
	}

	if _, err = f.Write(append(msg, "\n"...)); err != nil {
		panic(err)
	}

	c.FailOnError(err, "Error in write log")
}

func Flock(socketName string) {
	file, err := os.OpenFile(socketName, os.O_CREATE+os.O_APPEND, 0666)

	if err != nil {
		panic(fmt.Sprintf("%s: %s", "Error in creating lock file", err))
	}

	fd := file.Fd()
	err = syscall.Flock(int(fd), syscall.LOCK_EX+syscall.LOCK_NB)
	if err != nil {
		panic(fmt.Sprintf("%s: %s", "Error in lock file", err))
	}
}
