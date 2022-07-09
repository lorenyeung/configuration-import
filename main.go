package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lorenyeung/configuration-import/access"
	"github.com/lorenyeung/configuration-import/auth"
	"github.com/lorenyeung/configuration-import/helpers"

	log "github.com/sirupsen/logrus"
)

func main() {
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	startTime := time.Now()

	flags := helpers.SetFlags()
	helpers.SetLogger(flags.LogLevelVar)

	stringFlags := map[string]string{"-user": flags.UsernameVar, "-apikey": flags.ApikeyVar, "-url": flags.URLVar, "-securityJSONFile": flags.SecurityJSONFileVar}

	for i := range stringFlags {
		if stringFlags[i] == "" {
			log.Error(i + " cannot be empty")
		}
	}

	var creds auth.Creds
	creds.Username = flags.UsernameVar
	creds.Apikey = flags.ApikeyVar
	creds.URL = flags.URLVar

	//use different users to create things
	credsFilelength := 0
	credsFileHash := make(map[int][]string)
	if flags.CredsFileVar != "" {
		credsFile, err := os.Open(flags.CredsFileVar)
		if err != nil {
			log.Error("Invalid creds file:", err)
			os.Exit(1)
		}
		defer credsFile.Close()
		scanner := bufio.NewScanner(credsFile)

		for scanner.Scan() {
			credsFileCreds := strings.Split(scanner.Text(), " ")
			credsFileHash[credsFilelength] = credsFileCreds
			credsFilelength = credsFilelength + 1
		}

		flags.UsernameVar = credsFileHash[0][0]
		flags.ApikeyVar = credsFileHash[0][1]
		log.Info("Number of creds in file:", credsFilelength)
		log.Info("choose first one first:", flags.UsernameVar)
	}

	credCheck, err := auth.VerifyAPIKey(flags.URLVar, flags.UsernameVar, flags.ApikeyVar, flags)
	if !credCheck || err != nil {
		log.Error("Looks like there's an issue with checking your credentials. Exiting due to:", err)
		os.Exit(1)
	}

	//case switch for different access types
	workQueue := list.New()
	requestQueue := list.New()
	failureQueue := list.New()

	//hardcode for now
	go func() {
		err := access.ReadReposJSON(workQueue, flags, flags.NumReposVar)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	//work queue
	var ch = make(chan interface{}, flags.WorkersVar+1)
	var wg sync.WaitGroup
	for i := 0; i < flags.WorkersVar; i++ {
		go func(i int) {
			for {

				s, ok := <-ch
				if !ok {
					log.Info("Worker being returned to queue", i)
					wg.Done()
				}

				log.Debug("worker ", i, " starting job")

				if flags.CredsFileVar != "" {
					//pick random user and password from list
					numCreds := len(credsFileHash)
					rand.Seed(time.Now().UnixNano())
					randCredIndex := rand.Intn(numCreds)
					creds.Username = credsFileHash[randCredIndex][0]
					creds.Apikey = credsFileHash[randCredIndex][1]
				}

				//get data
				requestData := s.(access.ListTypes)
				log.Debug("data:", requestData.AccessType)
				switch requestData.AccessType {
				case "local":
					var RepoData access.RepositoryCreation
					RepoData.PackageType = requestData.PkgType
					RepoData.Rclass = "local"

					localData, err := json.Marshal(RepoData)
					if err != nil {
						log.Error(err)
					}
					resp, respCode, _, errorCode := auth.GetRestAPI("PUT", true, creds.URL+"/api/repositories/"+flags.PrefixVar+"-"+timestamp+"-"+requestData.PkgType+"-"+strconv.Itoa(requestData.Number)+"-local", creds.Username, creds.Apikey, "", localData, map[string]string{"Content-Type": "application/json"}, 0, flags, nil)
					if respCode != 200 {
						log.Error(string(resp), errorCode)
					}
				case "remote":
					var RepoData access.RepositoryCreation
					RepoData.PackageType = requestData.PkgType
					RepoData.Rclass = "remote"
					RepoData.URL = "https://" + flags.PrefixVar + "-" + timestamp + "-remote.com"

					remoteData, err := json.Marshal(RepoData)
					if err != nil {
						log.Error(err)
					}
					resp, respCode, _, errorCode := auth.GetRestAPI("PUT", true, creds.URL+"/api/repositories/"+flags.PrefixVar+"-"+timestamp+"-"+requestData.PkgType+"-"+strconv.Itoa(requestData.Number)+"-remote", creds.Username, creds.Apikey, "", remoteData, map[string]string{"Content-Type": "application/json"}, 0, flags, nil)
					if respCode != 200 {
						log.Error(string(resp), errorCode)
					}
				case "virtual":
					log.Error("VIRTUAL")
					var RepoData access.RepositoryCreation
					RepoData.PackageType = requestData.PkgType
					RepoData.Rclass = "local"

					localData, err := json.Marshal(RepoData)
					if err != nil {
						log.Error(err)
					}
					resp, respCode, _, errorCode := auth.GetRestAPI("PUT", true, creds.URL+"/api/repositories/"+flags.PrefixVar+"-"+timestamp+"-"+requestData.PkgType+"-"+strconv.Itoa(requestData.Number)+"-local", creds.Username, creds.Apikey, "", localData, map[string]string{"Content-Type": "application/json"}, 0, flags, nil)
					if respCode != 200 {
						log.Error(string(resp), errorCode)
					}

					var RepoDataVirtual access.RepositoryCreation
					RepoDataVirtual.PackageType = requestData.PkgType
					RepoDataVirtual.Rclass = "virtual"
					RepoDataVirtual.Repositories = []string{flags.PrefixVar + "-" + timestamp + "-" + requestData.PkgType + "-" + strconv.Itoa(requestData.Number) + "-local"}
					RepoDataVirtual.ExternalDependenciesEnabled = false

					//TODO BUGGED
					// VirtualData, err2 := json.Marshal(RepoDataVirtual)
					// if err2 != nil {
					// 	log.Error(err)
					// }
					// respVirt, respCodeVirt, _, errorCodeVirt := auth.GetRestAPI("PUT", true, creds.URL+"/api/repositories/"+flags.PrefixVar+"-"+timestamp+"-"+requestData.PkgType+"-"+strconv.Itoa(requestData.Number)+"-virtual", creds.Username, creds.Apikey, "", VirtualData, map[string]string{"Content-Type": "application/json"}, 0, flags, nil)
					// //if respCodeVirt != 200 {
					// log.Error(string(respVirt), errorCodeVirt, respCodeVirt)
					//}
				case "federated":
					var RepoData access.RepositoryCreation
					RepoData.PackageType = requestData.PkgType
					RepoData.Rclass = "local"

					localData, err := json.Marshal(RepoData)
					if err != nil {
						log.Error(err)
					}
					resp, respCode, _, errorCode := auth.GetRestAPI("PUT", true, creds.URL+"/api/repositories/"+flags.PrefixVar+"-"+timestamp+"-"+requestData.PkgType+"-"+strconv.Itoa(requestData.Number)+"-federated", creds.Username, creds.Apikey, "", localData, map[string]string{"Content-Type": "application/json"}, 0, flags, nil)
					if respCode != 200 {
						log.Error(string(resp), errorCode)
					}
				case "end":
					_, _, _, getErr := auth.GetRestAPI("GET", true, creds.URL+"/api/system/ping", creds.Username, creds.Apikey, "", nil, nil, 0, flags, nil)
					if getErr != nil {
						//well this is awkward
						log.Error("Something has gone wrong at the very end")
						os.Exit(1)
					}
					waiting := time.Now()
					for requestQueue.Len() > 0 {
						log.Info("End detected, waiting for last few requests to go through. Request queue size ", requestQueue.Len())
						time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)

						waitingSub := time.Now().Sub(waiting)
						if waitingSub > time.Duration(60)*time.Second {
							fmt.Println("Do you want to break manually? (y/n)")
							if askForConfirmation() {
								for requestQueue.Len() > 0 {
									requestQueue.Remove(requestQueue.Front())
								}
							}
						}

					}
					endTime := time.Now()
					log.Info("Completed import in ", endTime.Sub(startTime), "")
					if failureQueue.Len() > 0 {
						log.Warn("There were ", failureQueue.Len(), " failures. The following imports failed:")
						for e := failureQueue.Front(); e != nil; e = e.Next() {
							value := e.Value.(access.ListTypes)
							switch value.AccessType {
							case "group":

							case "user":

							}
						}
						fmt.Println("Do you want to retry these? (y/n)")
						if askForConfirmation() {
							for failureQueue.Len() > 0 {
								value := failureQueue.Front().Value.(access.ListTypes)
								log.Info("Re-queuing ", value.AccessType, " ", value)

								workQueue.PushBack(value)
								failureQueue.Remove(failureQueue.Front())
							}
							var endTask access.ListTypes
							endTask.AccessType = "end"
							workQueue.PushBack(endTask)
						} else {
							log.Info("Completed import in ", endTime.Sub(startTime), "")
							os.Exit(0)
						}
					} else {
						os.Exit(0)
					}

				}
				log.Debug("worker ", i, " finished job")
			}
		}(i)
	}

	//debug port
	go func() {
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()
	for {
		var count0 = 0
		for workQueue.Len() == 0 {
			log.Debug(" work queue is empty, sleeping for ", flags.WorkerSleepVar, " seconds...")
			time.Sleep(time.Duration(flags.WorkerSleepVar) * time.Second)
			count0++
			if count0 > 10 {
				log.Debug("Looks like nothing's getting put into the workqueue. You might want to enable -debug and take a look")
			}
			if workQueue.Len() > 0 {
				count0 = 0
			}
		}
		s := workQueue.Front().Value
		workQueue.Remove(workQueue.Front())
		ch <- s
	}
	close(ch)
	wg.Wait()
}

//Test if remote repository exists and is a remote
// func checkTypeAndRepoParams(creds auth.Creds, repoVar string) (string, string, string, string) {
// 	repoCheckData, repoStatusCode, _ := auth.GetRestAPI("GET", true, creds.URL+"/api/repositories/"+repoVar, creds.Username, creds.Apikey, "", nil, nil, 1, flags)
// 	if repoStatusCode != 200 {
// 		log.Error("Repo", repoVar, "does not exist.")
// 		os.Exit(0)
// 	}
// 	var result map[string]interface{}
// 	json.Unmarshal([]byte(repoCheckData), &result)
// 	//TODO: hard code for now, mass upload of files
// 	if result["rclass"] == "local" && result["packageType"].(string) == "generic" {
// 		return result["packageType"].(string), "", "", ""
// 	} else if result["rclass"] != "remote" {
// 		log.Error(repoVar, "is a", result["rclass"], "repository and not a remote repository.")
// 		//maybe here.
// 		os.Exit(0)
// 	}
// 	if result["packageType"].(string) == "pypi" {
// 		return result["packageType"].(string), result["url"].(string), result["pyPIRegistryUrl"].(string), result["pyPIRepositorySuffix"].(string)
// 	}
// 	return result["packageType"].(string), result["url"].(string), "", ""
// }

//https://gist.github.com/albrow/5882501
func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
}

func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}
