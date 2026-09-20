package api

import (
	"context"
	"net/http"
	"weatherservice/internal/application"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const (
	NumberOfMaxVideos = 10
)

type YoutubeAPI struct {
	client *http.Client

	key string
}

func NewYoutubeAPI(key string) *YoutubeAPI {
	return &YoutubeAPI{
		client: http.DefaultClient,
		key:    key,
	}
}

func (api *YoutubeAPI) FetchReportData(ctx context.Context, query ...string) (*application.DataToReport[application.VideosStream], error) {
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey(api.key))
	if err != nil {
		return nil, err
	}

	call := youtubeService.Search.List([]string{"snippet"}).Q(query[0]).MaxResults(NumberOfMaxVideos)

	response, err := call.Do()
	if err != nil {
		return nil, err
	}

	var videoStreams application.VideosStream
	for _, item := range response.Items {
		videoStreams = append(videoStreams, application.GeneralVideoStreamInfo{
			Title:   item.Snippet.Title,
			VideoID: item.Id.VideoId,
		})
	}

	//filter out videos that are not embeddable
	videoStreams = filterEmbeddableVideos(youtubeService, videoStreams)

	return &application.DataToReport[application.VideosStream]{
		Data: videoStreams,
	}, nil
}

func (api *YoutubeAPI) FetchGeneralInfo(ctx context.Context, _ ...string) (*application.DataToReport[application.VideosStream], error) {
	return nil, nil
}

func filterEmbeddableVideos(youtubeService *youtube.Service, videoStreams application.VideosStream) application.VideosStream {
	videoIDs := filterOutVideosIds(videoStreams)

	// Retrieve video statuses
	videoStatuses := youtubeService.Videos.List([]string{"status"}).Id(videoIDs...)

	videoStatusesResp, err := videoStatuses.Do()
	if err != nil {
		return nil
	}

	// Filter out videos that are not embeddable
	return filterOutUnavailableVideos(videoStatusesResp, videoStreams)
}

func filterOutVideosIds(videoStreams application.VideosStream) []string {
	var videoIDs []string
	for _, item := range videoStreams {
		videoIDs = append(videoIDs, item.VideoID)
	}

	return videoIDs
}

func filterOutUnavailableVideos(videos *youtube.VideoListResponse, videosStream application.VideosStream) application.VideosStream {
	var videosFiltered application.VideosStream

	for _, video := range videos.Items {
		for _, vid := range videosStream {
			if video.Status.Embeddable && video.Id == vid.VideoID {
				videosFiltered = append(videosFiltered, vid)
			}
		}
	}

	return videosFiltered
}
