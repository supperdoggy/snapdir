# snapdir
cli tool that creates a snapshot of the project and creates template config that can restore the project including folders structure and files data

## how to use
1. clone
`go run main.go clone /path/to/project project.json`

2. restore 
`go run main.go restore project.json /path/to/new/location`