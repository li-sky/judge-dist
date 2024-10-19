package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

type TestCase struct {
	Num    int    `json:"num"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

type Problem struct {
	ID        string     `json:"_id"`
	TestCases []TestCase `json:"testcases"`
}

type CompileTask struct {
	Token   string  `json:"token"`
	ID      int     `json:"test_case_id"`
	Problem Problem `json:"problem"`
}

type EvaluateTask struct {
	Token    string   `json:"token"`
	ID       int      `json:"test_case_id"`
	TestCase TestCase `json:"test_case"`
}

var cheval chan EvaluateTask
var chcomp chan CompileTask

func judge(num int) {
	os.Mkdir("judge/"+strconv.Itoa(num), 0777)
	for {
		// fetch a task from the channel
		task := <-cheval
		log.Printf("Evaluating task %+v\n", task.Token)
		// update database
		db.Exec("UPDATE evaluation_records SET test_case_status = 11 WHERE id = $1", task.ID)
		os.RemoveAll("judge/" + strconv.Itoa(num) + "/*")
		// copy input and program to judge/num
		srcFile := fmt.Sprintf("compile/%s.out", task.Token)
		dstFile := fmt.Sprintf("judge/%d/file.out", num)
		// copy with cp command
		cmd := exec.Command("cp", srcFile, dstFile)
		err := cmd.Run()
		if err != nil {
			log.Fatalf("Failed to copy executable: %v\n", err)
		}
		if task.TestCase.Input != "" {
			srcfile := task.TestCase.Input
			dstfile := fmt.Sprintf("judge/%d/input.txt", num)
			cmd = exec.Command("cp", srcfile, dstfile)
			err = cmd.Run()
			if err != nil {
				log.Fatalf("Failed to copy input: %v\n", err)
			}
		} else {
			os.Create(fmt.Sprintf("judge/%d/input.txt", num))
		}
		// run program
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v\n", err)
		}
		cmd = exec.Command("docker", "run", "--rm", "-m", "512m", "--mount",
			fmt.Sprintf("type=bind,source=%s/judge/%d/,target=/data/", pwd, num), "judge:latest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				log.Printf("Command exited with non-zero status: %v\n", exitErr.ExitCode())
				if exitErr.ExitCode() == 124 {
					// time limit exceeded
					db.Exec("UPDATE evaluation_records SET test_case_status = 5 WHERE id = $1", task.ID)
				} else {
					db.Exec("UPDATE evaluation_records SET test_case_status = 4 WHERE id = $1", task.ID)
				}
				log.Printf("Command output: %s\n", string(output))
			} else {
				log.Printf("Failed to run command: %v\n", err)
				db.Exec("UPDATE evaluation_records SET test_case_status = 9 WHERE id = $1", task.ID)
			}
		}
		// compare output
		cmd = exec.Command("checker/noip-checker", task.TestCase.Input, task.TestCase.Output, fmt.Sprintf("judge/%d/output.txt", num))
		err = cmd.Run()
		if err != nil {
			// wrong answer
			db.Exec("UPDATE evaluation_records SET test_case_status = 8 WHERE id = $1", task.ID)
		} else {
			// Accepted
			db.Exec("UPDATE evaluation_records SET test_case_status = 1 WHERE id = $1", task.ID)
		}
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func comp(num int) {
	os.Mkdir("compile/"+strconv.Itoa(num), 0777)
	for {
		// fetch a task from the channel
		task := <-chcomp
		log.Printf("Compiling task %+v\n", task.Token)
		// update database
		db.Exec("UPDATE evaluation_records SET test_case_status = 10 WHERE id = $1", task.ID)
		// compile the code
		// copy from submissions to compile
		srcFile := fmt.Sprintf("submissions/%s.cpp", task.Token)
		dstFile := fmt.Sprintf("compile/%d/file.cpp", num)
		err := copyFile(srcFile, dstFile)
		if err != nil {
			log.Printf("Failed to copy file: %v\n", err)
		}
		// compile the code
		path, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v\n", err)
		}
		fmt.Print(path)
		cmd := exec.Command("docker", "run", "--rm", "--mount",
			fmt.Sprintf("type=bind,source=%s/compile/%d/,target=/data/", path, num), "compile:latest")
		output, err := cmd.CombinedOutput()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				log.Printf("Command exited with non-zero status: %v\n", exitErr.ExitCode())
				if exitErr.ExitCode() == 124 {
					// compile time out
					db.Exec("UPDATE evaluation_records SET test_case_status = 3 WHERE id = $1", task.ID)
				} else {
					db.Exec("UPDATE evaluation_records SET test_case_status = 2 WHERE id = $1", task.ID)
				}
				log.Printf("Command output: %s\n", string(output))
			} else {
				log.Printf("Failed to run command: %v\n", err)
				db.Exec("UPDATE evaluation_records SET test_case_status = 9 WHERE id = $1", task.ID)
			}
		} else {
			// compilation successful
			// move executable from compile/num to compile/{token}.out with mv command
			cmd := exec.Command("mv", fmt.Sprintf("compile/%d/a.out", num), fmt.Sprintf("compile/%s.out", task.Token))
			err := cmd.Run()
			if err != nil {
				log.Fatalf("Failed to copy executable: %v\n", err)
			}
			db.Exec("UPDATE evaluation_records SET test_case_status = 12 WHERE id = $1", task.ID)
			evtask0 := EvaluateTask{Token: task.Token, ID: task.ID, TestCase: task.Problem.TestCases[0]}
			cheval <- evtask0
			for i, tc := range task.Problem.TestCases {
				if i == 0 {
					continue
				}
				var lastInsertID int
				err = db.QueryRow("INSERT INTO evaluation_records (token, test_case_id, test_case_status, question_id) VALUES ($1, $2, $3, $4) RETURNING id",
					task.Token, i, 12, task.Problem.ID).Scan(&lastInsertID)
				if err != nil {
					log.Fatalf("Failed to insert evaluation record: %v\n", err)
					return
				}
				evtask := EvaluateTask{Token: task.Token, ID: lastInsertID, TestCase: tc}
				cheval <- evtask
			}
		}
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// read testcases from file testcases.json

	file, err := os.Open("testcases.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var problems []Problem
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&problems)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully parsed testcases.json")
	fmt.Printf("%+v\n", problems)

	connstr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err = sql.Open("postgres", connstr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to PostgreSQL")

	// create channel for evaluating tasks
	chcomp = make(chan CompileTask, 100)
	cheval = make(chan EvaluateTask, 100)

	// start judge workers
	runnerCount, err := strconv.Atoi(os.Getenv("RUNNER_COUNT"))
	if err != nil {
		log.Fatalf("Failed to parse RUNNER_COUNT: %v\n", err)
	}

	compilerCount, err := strconv.Atoi(os.Getenv("COMPILER_COUNT"))
	if err != nil {
		log.Fatalf("Failed to parse COMPILER_COUNT: %v\n", err)
	}
	for i := 0; i < runnerCount; i++ {
		go judge(i)
	}

	for i := 0; i < compilerCount; i++ {
		go comp(i)
	}

	r := gin.Default()

	r.POST("/api/v1/submit", func(c *gin.Context) {
		var requestBody struct {
			Code string `json:"code"`
			ID   string `json:"_id"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		decodedCode, err := base64.StdEncoding.DecodeString(requestBody.Code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode base64 code"})
			return
		}

		// Find the problem with the given ID
		var problem Problem
		for _, p := range problems {
			if p.ID == requestBody.ID {
				problem = p
				break
			}
		}
		if problem.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "problem not found"})
			return
		}

		// add the test cases to the database

		token := uuid.New().String()

		// save the code to disk

		file, err := os.Create(fmt.Sprintf("submissions/%s.cpp", token))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer file.Close()

		_, err = file.Write(decodedCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// insert compile task into database
		var lastInsertID int
		err = db.QueryRow("INSERT INTO evaluation_records (token, test_case_id, test_case_status, question_id) VALUES ($1, $2, $3, $4) RETURNING id",
			token, problem.TestCases[0].Num, 0, problem.ID).Scan(&lastInsertID)
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		comptask := CompileTask{Token: token, ID: lastInsertID, Problem: problem}
		chcomp <- comptask

		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	r.GET("/api/v1/query", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
			return
		}

		rows, err := db.Query("SELECT test_case_id, test_case_status FROM evaluation_records WHERE token = $1", token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var response struct {
			Token     string `json:"token"`
			Testcases []struct {
				Num    int `json:"num"`
				Status int `json:"status"`
			} `json:"testcases"`
		}

		response.Token = token
		for rows.Next() {
			var num, status int
			err := rows.Scan(&num, &status)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			response.Testcases = append(response.Testcases, struct {
				Num    int `json:"num"`
				Status int `json:"status"`
			}{Num: num, Status: status})
		}
		c.JSON(http.StatusOK, response)
	})

	r.Run(":5050") // Run on port 8080
}
