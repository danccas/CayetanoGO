package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	k_means "./k_means"
	templates "./templates"
)

const ALLOWED_FILE_TYPE = "text/plain; charset=utf-8"

const (
	clusterNumberParam   = "clusters"
	iterationNumberParam = "iterations"
	distanceMethodParam  = "distance"
)

func paramToInt(param string) (int, error) {
	if param == "" {
		return 0, errors.New(" param is empty")
	}

	return strconv.Atoi(param)
}

func processHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseMultipartForm(32 << 20)

		clustersParam, err := paramToInt(r.Form.Get(clusterNumberParam))
		if err != nil {
			http.Error(w,
				clusterNumberParam+err.Error(),
				http.StatusBadRequest)
			return
		} else if clustersParam < 0 {
			http.Error(w,
				clusterNumberParam+" is negative or zero ",
				http.StatusBadRequest)
			return
		}

		iterationsParam, err := paramToInt(r.Form.Get(iterationNumberParam))
		if err != nil {
			http.Error(w,
				iterationNumberParam+err.Error(),
				http.StatusBadRequest)
			return
		} else if iterationsParam < 0 {
			http.Error(w,
				iterationNumberParam+" is negative or zero ",
				http.StatusBadRequest)
			return
		}

		var distanceParam = r.Form.Get(distanceMethodParam)
		var distanceMethod k_means.DistanceFunc

		switch distanceParam {
		case k_means.EuclidDistance:
			distanceMethod = k_means.EuclidDistanceFunc
		default:
			http.Error(w,
				distanceMethodParam+" is unknown",
				http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w,
				err.Error(),
				http.StatusInternalServerError)

			return
		}

		defer file.Close()

		var Buf bytes.Buffer
		io.Copy(&Buf, file)

		if http.DetectContentType(Buf.Bytes()) != ALLOWED_FILE_TYPE {
			http.Error(w,
				http.StatusText(http.StatusUnsupportedMediaType),
				http.StatusUnsupportedMediaType)
			return
		}

		var claims Claims
		err = json.Unmarshal(Buf.Bytes(), &claims)
		if err != nil {
			http.Error(w,
				err.Error(),
				http.StatusInternalServerError)

			return
		}

		Buf.Reset()

		points, err := ClaimsToPoints(claims)
		if err != nil {
			http.Error(w,
				err.Error(),
				http.StatusInternalServerError)

			return
		}

		log.Println("new data request accepted...")

		cls, err := k_means.Calc(points, int32(clustersParam), int32(iterationsParam), distanceMethod)
		if err != nil {
			http.Error(w,
				err.Error(),
				http.StatusInternalServerError)
			return
		}

		data, err := json.MarshalIndent(&cls, "  ", "    ")
		if err != nil {
			http.Error(w,
				err.Error(),
				http.StatusInternalServerError)
			return
		}

		template, err := templates.GetTemplateWithData("map.html", struct{ Data string }{Data: string(data)})
		if err != nil {
			http.Error(w,
				err.Error(),
				http.StatusInternalServerError)
			return
		}

		w.Write(template)
	} else {
		http.Error(w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
	}
}

func Start() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/form.html")
	})

	http.HandleFunc("/analizar", processHandler)

	err := http.ListenAndServe(":8010", nil)
	if err != nil {
		log.Fatal("FATAL:", err.Error())
	}
}
