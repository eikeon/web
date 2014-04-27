package web

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path"
)

func AddResources(root string) {
	if j, err := os.OpenFile(path.Join(root, "resources.json"), os.O_RDONLY, 0666); err == nil {
		dec := json.NewDecoder(j)
		for {
			var resources []Resource
			if err := dec.Decode(&resources); err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}
			for _, r := range resources {
				Put(&r)
			}
		}
	} else {
		log.Println(err)
	}

}
