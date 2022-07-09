package access

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"

	"github.com/lorenyeung/configuration-import/auth"
	"github.com/lorenyeung/configuration-import/helpers"

	log "github.com/sirupsen/logrus"
)

type ListTypes struct {
	AccessType string
	PkgType    string
	Number     int
}
type ArtifactoryVersion struct {
	Version  string   `json:"version"`
	Revision string   `json:"revision"`
	Addons   []string `json:"addons"`
	License  string   `json:"license"`
}

type RepositoryCreation struct {
	Rclass                      string   `json:"rclass"`
	PackageType                 string   `json:"packageType"`
	URL                         string   `json:"url,omitempty"`
	Repositories                []string `json:"repositories,omitempty"`
	ExternalDependenciesEnabled bool     `json:"externalDependenciesEnabled,omitempty"`
}

type ArtifactoryError struct {
	Errors []ArtifactoryErrorDetail `json:"errors"`
}

type ArtifactoryErrorDetail struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type Data struct {
	RepoTypes []string `json:"repotypes"`
}

func ReadReposJSON(workQueue *list.List, flags helpers.Flags, numRepos int) error {

	//get art version
	var artVer ArtifactoryVersion
	data, _, _, getErr := auth.GetRestAPI("GET", true, flags.URLVar+"/api/system/version", flags.UsernameVar, flags.ApikeyVar, "", nil, nil, 0, flags, nil)
	if getErr != nil {
		return getErr
	}
	err := json.Unmarshal(data, &artVer)
	if err != nil {
		return err
	}

	//TODO: this reads whole file into memory, be wary of OOM
	log.Info("reading repo types json")
	data, err = ioutil.ReadFile(flags.SecurityJSONFileVar)
	if err != nil {
		log.Error("Error reading security json" + err.Error() + " " + helpers.Trace().Fn + ":" + strconv.Itoa(helpers.Trace().Line))
		return errors.New("Error reading security json" + err.Error() + " " + helpers.Trace().Fn + ":" + strconv.Itoa(helpers.Trace().Line))
	}
	var RepoTypedata Data
	err = json.Unmarshal(data, &RepoTypedata)
	if err != nil {
		log.Error(err)
	}

	reasons := make([]string, 0)
	reasons = append(reasons,
		"local",
		"remote",
		"virtual",
		"federated")
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	for i := 0; i < numRepos; i++ {
		repo := reasons[rand.Intn(len(reasons))]
		pkg := RepoTypedata.RepoTypes[rand.Intn(len(RepoTypedata.RepoTypes))]
		fmt.Println("repo:", repo, "pkg", pkg)
		var RepoTask ListTypes
		RepoTask.AccessType = repo
		RepoTask.PkgType = pkg
		RepoTask.Number = i
		//md := "{\"rclass\" : \"local\",\"packageType\" : \"" + requestData.PkgType + "\"}"
		workQueue.PushBack(RepoTask)
	}

	var endTask ListTypes
	endTask.AccessType = "end"
	workQueue.PushBack(endTask)
	return nil
}
