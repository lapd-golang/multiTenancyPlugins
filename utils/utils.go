package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/gorilla/mux"
	"github.com/multiTenancyPlugins/headers"
)

type ValidationOutPutDTO struct {
	ContainerID  string
	Links        []string
	VolumesFrom  []string
	Binds        []string
	Env          []string
	ErrorMessage string
	//Quota can live here too? Currently quota needs only raise error
	//What else
}

//UTILS

func ModifyRequest(r *http.Request, body io.Reader, urlStr string, containerID string) (*http.Request, error) {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
		r.Body = rc
	}
	if urlStr != "" {
		u, err := url.Parse(urlStr)

		if err != nil {
			return nil, err
		}
		r.URL = u
		mux.Vars(r)["name"] = containerID
	}
	return r, nil
}

//Assumes ful ID was injected
func IsResourceOwner(cluster cluster.Cluster, tenantID string, resourceId string, resourceType string) bool {
	switch resourceType {
	case "container":
		for _, container := range cluster.Containers() {
			if container.Info.ID == resourceId {
				return container.Labels[headers.TenancyLabel] == tenantID
			}
		}
		return false
	case "network":
		for _, network := range cluster.Networks() {
			if network.ID == resourceId {
				return strings.HasPrefix(network.Name, ConstructNetworkPrefix(tenantID))
			}
		}
		return false
	default:
		log.Warning("Unsupported resource type for authorization.")
		return false
	}
}

//Verify exec id is in a container on the tennant id
func VerifyExecContainerTenant(cluster cluster.Cluster, tenantId string, r *http.Request) bool {
	for _, container := range cluster.Containers() {
		for _, execID := range container.Info.ExecIDs {
			if execID == mux.Vars(r)["execid"] { //getExecId(r) {
				return container.Labels[headers.TenancyLabel] == tenantId
			}
		}
	}
	return false
}

//Expand / Refactor
func CleanUpLabeling(r *http.Request, rec *httptest.ResponseRecorder) []byte {
	newBody := bytes.Replace(rec.Body.Bytes(), []byte(headers.TenancyLabel), []byte(" "), -1)
	//TODO - Here we just use the token for the tenant name for now so we remove it from the data before returning to user.
	newBody = bytes.Replace(newBody, []byte(r.Header.Get(headers.AuthZTenantIdHeaderName)), []byte(""), -1)
	newBody = bytes.Replace(newBody, []byte(",\" \":\" \""), []byte(""), -1)
	log.Debugf("Clean up labeling done.")
	//	log.Debug("Got this new body...", string(newBody))
	return newBody
}

// RandStringBytesRmndr used to generate a name for docker volume create when no name is supplied
// The tenant id is then appended to the name by the caller
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

type CommandEnum string

const (
	//For reference look at primary.go
	PING    CommandEnum = "ping"
	EVENTS  CommandEnum = "events"
	INFO    CommandEnum = "info"
	VERSION CommandEnum = "version"
	//SKIP ...

	PS   CommandEnum = "containersps"
	JSON CommandEnum = "containersjson"

	CONTAINER_ARCHIVE CommandEnum = "containerarchive"
	CONTAINER_EXPORT  CommandEnum = "containerexport"
	CONTAINER_IMPORT  CommandEnum = "containerimport"
	CONTAINER_CHANGES CommandEnum = "containerchanges"
	CONTAINER_JSON    CommandEnum = "containerjson"
	CONTAINER_TOP     CommandEnum = "containertop"
	CONTAINER_LOGS    CommandEnum = "containerlogs"
	CONTAINER_STATS   CommandEnum = "containerstats"
	//SKIP ...
	NETWORKS_LIST      CommandEnum = "networkslist"
	NETWORK_INSPECT    CommandEnum = "networkinspect"
	NETWORK_CONNECT    CommandEnum = "networkconnect"
	NETWORK_DISCONNECT CommandEnum = "networkdisconnect"
	NETWORK_CREATE     CommandEnum = "networkcreate"
	NETWORK_DELETE     CommandEnum = "networkdelete"
	//SKIP ...
	VOLUMES_LIST   CommandEnum = "volumeslist"
	VOLUME_INSPECT CommandEnum = "volumeinspect"
	VOLUME_CREATE  CommandEnum = "volumecreate"
	VOLUME_DELETE  CommandEnum = "volumedelete"

	//SKIP ...
	//POST
	CONTAINER_CREATE  CommandEnum = "containerscreate"
	CONTAINER_KILL    CommandEnum = "containerkill"
	CONTAINER_PAUSE   CommandEnum = "containerpause"
	CONTAINER_UNPAUSE CommandEnum = "containerunpause"
	CONTAINER_RENAME  CommandEnum = "containerrename"
	CONTAINER_RESTART CommandEnum = "containerrestart"
	CONTAINER_START   CommandEnum = "containerstart"
	CONTAINER_STOP    CommandEnum = "containerstop"
	CONTAINER_UPDATE  CommandEnum = "containerupdate"
	CONTAINER_WAIT    CommandEnum = "containerwait"
	CONTAINER_RESIZE  CommandEnum = "containerresize"
	CONTAINER_ATTACH  CommandEnum = "containerattach"
	CONTAINER_COPY    CommandEnum = "containercopy"
	CONTAINER_EXEC    CommandEnum = "containerexec"
	EXEC_START        CommandEnum = "execstart"
	EXEC_RESIZE       CommandEnum = "execresize"
	EXEC_JSON         CommandEnum = "execjson"
	//SKIP ...

	CONTAINER_DELETE CommandEnum = "containerdelete"

	IMAGES_JSON   CommandEnum = "imagesjson"
	IMAGE_PULL    CommandEnum = "imagescreate"
	IMAGE_SEARCH  CommandEnum = "imagessearch"
	IMAGE_JSON    CommandEnum = "imagejson"
	IMAGE_HISTORY CommandEnum = "imagehistory"
)

var invMapmap map[string]CommandEnum
var initialized bool

func ParseCommand(r *http.Request) CommandEnum {
	if !initialized {
		invMapmap = make(map[string]CommandEnum)
		invMapmap["ping"] = PING
		invMapmap["events"] = EVENTS
		invMapmap["info"] = INFO
		invMapmap["version"] = VERSION
		//SKIP ...
		invMapmap["containersps"] = PS
		invMapmap["containersjson"] = JSON
		invMapmap["containerarchive"] = CONTAINER_ARCHIVE
		invMapmap["containerexport"] = CONTAINER_EXPORT
		invMapmap["containerimport"] = CONTAINER_IMPORT
		invMapmap["containerchanges"] = CONTAINER_CHANGES
		invMapmap["containerjson"] = CONTAINER_JSON
		invMapmap["containertop"] = CONTAINER_TOP
		invMapmap["containerlogs"] = CONTAINER_LOGS
		invMapmap["containerstats"] = CONTAINER_STATS
		//SKIP ...
		invMapmap["networkslist"] = NETWORKS_LIST
		invMapmap["networkinspect"] = NETWORK_INSPECT
		invMapmap["networkconnect"] = NETWORK_CONNECT
		invMapmap["networkdisconnect"] = NETWORK_DISCONNECT
		invMapmap["networkcreate"] = NETWORK_CREATE
		invMapmap["networkdelete"] = NETWORK_DELETE
		//SKIP ...
		invMapmap["volumeslist"] = VOLUMES_LIST
		invMapmap["volumeinspect"] = VOLUME_INSPECT
		invMapmap["volumecreate"] = VOLUME_CREATE
		invMapmap["volumedelete"] = VOLUME_DELETE
		//POST
		invMapmap["containerscreate"] = CONTAINER_CREATE
		invMapmap["containerkill"] = CONTAINER_KILL
		invMapmap["containerpause"] = CONTAINER_PAUSE
		invMapmap["containerunpause"] = CONTAINER_UNPAUSE
		invMapmap["containerrename"] = CONTAINER_RENAME
		invMapmap["containerrestart"] = CONTAINER_RESTART
		invMapmap["containerstart"] = CONTAINER_START
		invMapmap["containerstop"] = CONTAINER_STOP
		invMapmap["containerupdate"] = CONTAINER_UPDATE
		invMapmap["containerwait"] = CONTAINER_WAIT
		invMapmap["containerresize"] = CONTAINER_RESIZE
		invMapmap["containerattach"] = CONTAINER_ATTACH
		invMapmap["containercopy"] = CONTAINER_COPY
		invMapmap["containerexec"] = CONTAINER_EXEC
		invMapmap["execstart"] = EXEC_START
		invMapmap["execresize"] = EXEC_RESIZE
		invMapmap["execjson"] = EXEC_JSON
		//SKIP ...
		invMapmap["containerdelete"] = CONTAINER_DELETE

		invMapmap["imagesjson"] = IMAGES_JSON
		invMapmap["imagescreate"] = IMAGE_PULL
		invMapmap["imagessearch"] = IMAGE_SEARCH
		invMapmap["imagejson"] = IMAGE_JSON
		invMapmap["imagehistory"] = IMAGE_HISTORY
		initialized = true
	}
	return invMapmap[commandParser(r)]
}

var containersRegexp = regexp.MustCompile("/containers/(.*)/(.*)|/containers/(\\w+)")
var networksRegexp = regexp.MustCompile("/networks/(.*)/(.*)|/networks/(\\w+)")
var volumesRegexp = regexp.MustCompile("/volumes/(.*)/(.*)|/volumes/(\\w+)")
var clusterRegExp = regexp.MustCompile("/(.*)/(.*)")
var imagesRegexp = regexp.MustCompile("/images/(.*)/(.*)|/images/(\\w+)")

func commandParser(r *http.Request) string {
	containersParams := containersRegexp.FindStringSubmatch(r.URL.Path)
	networksParams := networksRegexp.FindStringSubmatch(r.URL.Path)
	volumesParams := volumesRegexp.FindStringSubmatch(r.URL.Path)
	clusterParams := clusterRegExp.FindStringSubmatch(r.URL.Path)
	imagesParams := imagesRegexp.FindStringSubmatch(r.URL.Path)

	log.Debug(containersParams)
	log.Debug(networksParams)
	log.Debug(clusterParams)
	log.Debug(imagesParams)
	log.Debug(volumesParams)

	switch r.Method {
	case "DELETE":
		if len(containersParams) > 0 {
			return "containerdelete"
		}
		if len(networksParams) > 0 {
			return "networkdelete"
		}
		if len(imagesParams) > 0 {
			return "imagedelete"
		}
		if len(volumesParams) > 0 {
			return "volumedelete"
		}

	case "GET", "POST":
		if len(containersParams) == 4 && containersParams[2] != "" {
			log.Debug("A1")
			return "container" + containersParams[2]
		} else if len(containersParams) == 4 && containersParams[3] != "" {
			log.Debug("A2")
			return "containers" + containersParams[3] //S
		}
		if len(imagesParams) == 4 && imagesParams[2] != "" {
			log.Debug("A1")
			return "image" + imagesParams[2]
		} else if len(imagesParams) == 4 && imagesParams[3] != "" {
			log.Debug("A2")
			log.Debug("images" + imagesParams[3])
			return "images" + imagesParams[3] //S
		}
		if strings.HasSuffix(r.URL.Path, "/networks") {
			return "networkslist"
		}
		if len(networksParams) == 4 && networksParams[3] != "" {
			if networksParams[3] == "create" {
				return "networkcreate"
			}
			return "networkinspect"
		} else if len(networksParams) == 4 {
			return "network" + networksParams[2]
		}

		if strings.HasSuffix(r.URL.Path, "/volumes") ||
			strings.HasSuffix(r.URL.Path, "/volumes/") ||
			strings.Contains(r.RequestURI, "/volumes?") {
			return "volumeslist"
		}
		if len(volumesParams) == 4 {
			if volumesParams[3] == "create" {
				return "volumecreate"
			}
			return "volumeinspect"
		}

		if len(clusterParams) == 3 && (clusterParams[2] == "start" || clusterParams[2] == "resize" || clusterParams[2] == "json") {
			log.Debug("A3")
			return "exec" + clusterParams[2]
		} else {
			log.Debug("A3")
			return clusterParams[2]
		}
	}
	return "This is not supported yet and will end up in the default of the Switch"
}

//FilterNetworks - filter out all networks not created by tenant.
func FilterNetworks(r *http.Request, rec *httptest.ResponseRecorder) []byte {
	var networks cluster.Networks
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&networks); err != nil {
		log.Error(err)
		return nil
	}
	var candidates cluster.Networks
	namePrefix := ConstructNetworkPrefix(r.Header.Get(headers.AuthZTenantIdHeaderName))
	for _, network := range networks {
		fullName := strings.SplitN(network.Name, "/", 2)
		name := fullName[len(fullName)-1]
		if strings.HasPrefix(name, namePrefix) {
			network.Name = strings.TrimPrefix(name, namePrefix)
			candidates = append(candidates, network)
		}
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(candidates); err != nil {
		log.Error(err)
		return nil
	}
	return buf.Bytes()
}

func ConstructNetworkPrefix(tenantID string) string {
	// Network name prefix pattern: "s" + $(lowercase tenantID) + "-"
	return "s" + strings.ToLower(tenantID) + "-"
}

type ErrorInfo struct {
	Err    error
	Status int
}
