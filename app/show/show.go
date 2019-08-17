package show

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/zhikiri/uaitunes-podcasts/app/crawler"
	"github.com/zhikiri/uaitunes-podcasts/app/genre"
)

type Show struct {
	ID     int
	Name   string
	Artist string
	RSS    string
	Genres []int
	Image  ShowImage
}

type ShowImage struct {
	Small  string
	Medium string
	Big    string
}

type ShowRequestOptions struct {
	crawler.RequestOptions
	ShowDetailsURL string
}

type showDetailsResponse struct {
	Results []struct {
		CollectionId   int    `json:"collectionId"`
		ArtistId       int    `json:"artistId"`
		ArtistName     string `json:"artistName"`
		CollectionName string `json:"collectionName"`
		GenreIds       []int  `json:"genreIds"`
		ArtworkURL30   string `json:"artworkURL30"`
		ArtworkURL60   string `json:"artworkURL60"`
		ArtworkURL100  string `json:"artworkURL100"`
		FeedURL        string `json:"feedUrl"`
	} `json:"results"`
}

func ShowsRequestOptions(genre *genre.Genre) *ShowRequestOptions {

	return &ShowRequestOptions{
		RequestOptions: crawler.RequestOptions{
			LookupURL: genre.URL,
			Pattern:   "div[id=selectedcontent] .column a[href]",
		},
		ShowDetailsURL: "https://itunes.apple.com/lookup?id=",
	}
}

func GetShows(options *ShowRequestOptions) ([]*Show, []error) {

	shows := []*Show{}

	entities, err := crawler.GetEntities(&options.RequestOptions)
	if err != nil {
		return shows, []error{err}
	}

	errorsCh := make(chan error, len(entities))
	showsCh := make(chan *Show, len(entities))
	var wg sync.WaitGroup

	for _, url := range entities {

		id, err := crawler.GetEntityIDFromURL(url)
		if err != nil {
			return shows, []error{err}
		}

		wg.Add(1)
		go func() {
			show, err := getShowDetails(id, options)
			if err != nil {
				errorsCh <- err
			} else {
				showsCh <- show
			}
			wg.Done()
		}()
	}
	wg.Wait()
	close(showsCh)
	close(errorsCh)

	for show := range showsCh {
		shows = append(shows, show)
	}

	errors := []error{}
	for err := range errorsCh {
		errors = append(errors, err)
	}

	return shows, errors
}

func getShowDetails(id int, options *ShowRequestOptions) (*Show, error) {

	url := fmt.Sprintf("%s%d", options.ShowDetailsURL, id)
	resp, err := http.Get(url)
	if err != nil {
		return &Show{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Show{}, err
	}

	var details showDetailsResponse
	err = json.Unmarshal(body, &details)
	if err != nil {
		return &Show{}, err
	}
	res := details.Results[0]

	return &Show{
		ID:     res.CollectionId,
		Name:   res.CollectionName,
		Artist: res.ArtistName,
		RSS:    res.FeedURL,
		Genres: res.GenreIds,
		Image: ShowImage{
			Small:  res.ArtworkURL30,
			Medium: res.ArtworkURL60,
			Big:    res.ArtworkURL100,
		},
	}, nil
}
