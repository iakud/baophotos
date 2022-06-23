package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/iakud/baophotos/session"
)

const (
	UPLOAD_DIR = "./uploads"
)

func main() {
	if !isExists(UPLOAD_DIR) {
		if err := os.Mkdir(UPLOAD_DIR, 0666); err != nil {
			log.Fatalln(err)
		}
	}
	http.HandleFunc("/", listHandler)
	http.HandleFunc("/login", loginHandler)
	// http.HandleFunc("/", testHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/view", viewHandler)
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, err := template.ParseFiles("login.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t.Execute(w, nil)
	}
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Println(r.PostForm)
		if password := r.PostFormValue("password"); password != "220612" {
			log.Println("login:", password)
			http.Error(w, "password error", http.StatusInternalServerError)
			return
		}

		session := session.Start(w, r)
		session.Set("status", "OK")
		log.Println("login ok")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	session := session.Start(w, r)
	if session.Get("status") != "OK" {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	if r.Method == "GET" {
		t, err := template.ParseFiles("upload.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t.Execute(w, nil)
		return
	}
	if r.Method == "POST" {
		f, h, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			return
		}
		filename := h.Filename
		defer f.Close()
		t, err := os.Create(UPLOAD_DIR + "/" + filename)
		if err != nil {
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			return
		}
		defer t.Close()
		if _, err := io.Copy(t, f); err != nil {
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
func testHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "<h1>BAOBAO的相册</h1>")
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	session := session.Start(w, r)
	if session.Get("status") != "OK" {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	fileInfoArr, err := ioutil.ReadDir("./uploads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	locals := make(map[string]interface{})
	images := []string{}
	for _, fileInfo := range fileInfoArr {
		images = append(images, fileInfo.Name())
	}
	locals["images"] = images
	t, err := template.ParseFiles("list.html")
	if err != nil {
		http.Error(w, err.Error(),
			http.StatusInternalServerError)
		return
	}
	t.Execute(w, locals)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	session := session.Start(w, r)
	if session.Get("status") != "OK" {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	imageId := r.FormValue("id")
	imagePath := UPLOAD_DIR + "/" + imageId
	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, imagePath)
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}

func renderHtml(w http.ResponseWriter, tmpl string, locals map[string]interface{}) error {
	t, err := template.ParseFiles(tmpl + ".html")
	if err != nil {
		return err
	}
	return t.Execute(w, locals)
}
