package chorgtree

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

// Node is a type that contains Organization data as well as a list of references to children Nodes.
type Node struct {
	mux                  sync.Mutex // For locking Children Node array
	BusinessOrganization Organization
	Children             []*Node
}

// Organization is a type that contains an Organizations Name and ID, as well as a list of sub-Organizations.
type Organization struct {
	Name               string
	ID                 string
	SubOrganizationIds []string
	Environments       []*Environment
}

// Environment is a type that contains an Environemnt Name and ID.
type Environment struct {
	ID   string
	Name string
}

// Application is a type that contains an Application Domain, Full Domain, Status, and File Name.
type Application struct {
	Domain     string
	FullDomain string
	Status     string
	FileName   string
}

func errorCheck(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}

// InitTree initializes a new organization heirarchy tree.
func InitTree(root string, username string, password string) *Node {
	g := &sync.WaitGroup{}

	// Construct root Node
	byteArray := getOrganizationMetrics(root, username, password)
	var organization Organization
	json.Unmarshal(byteArray, &organization)
	node := &Node{BusinessOrganization: organization, Children: nil}

	// Build remaining Nodes
	g.Add(1)
	node.buildOrgTree(username, password, g)
	g.Done()

	return node
}

func (p *Node) buildOrgTree(username string, password string, g *sync.WaitGroup) {
	defer g.Done()
	for _, v := range p.BusinessOrganization.SubOrganizationIds {
		byteArray := getOrganizationMetrics(v, username, password)
		var organization Organization
		json.Unmarshal(byteArray, &organization)

		node := &Node{BusinessOrganization: organization, Children: nil}

		p.mux.Lock()
		p.Children = append(p.Children, node)
		p.mux.Unlock()

		g.Add(1)
		go node.buildOrgTree(username, password, g)
	}
}

func getOrganizationMetrics(orgID string, username string, password string) []byte {
	const organizationsEndpoint string = "https://anypoint.mulesoft.com/accounts/api/organizations/"
	requestURL := fmt.Sprintf("%s%s", organizationsEndpoint, orgID)

	client := &http.Client{}

	req, err := http.NewRequest("GET", requestURL, nil)
	errorCheck(err)
	req.SetBasicAuth(username, password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	errorCheck(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Non-OK HTTP status:", resp.StatusCode)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	errorCheck(err)

	return body
}
