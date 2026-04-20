package urls

import (
	"context"
	"fmt"
	"sync"

	"github.com/artni96/url-shortener/internal/model"
)

type Result struct {
	ShortURL string
	Err      error
}

func generator(doneCh chan struct{}, urls []model.URLDelete) chan model.URLDelete {
	inCh := make(chan model.URLDelete)

	go func() {
		defer close(inCh)

		for _, url := range urls {
			select {
			case <-doneCh:
				return
			case inCh <- url:

			}
		}
	}()
	return inCh
}

func canBeDeleted(ctx context.Context, doneCh chan struct{}, inCh chan model.URLDelete, repo *DBURLRepository) chan Result {
	res := make(chan Result)

	go func() {
		defer close(res)

		for url := range inCh {

			urlData := Result{
				ShortURL: url.ShortURL,
				Err:      nil,
			}
			var createdBy int
			selectQuery := "SELECT created_by FROM urls WHERE short_url = $1"
			err := repo.db.GetContext(ctx, &createdBy, selectQuery, url.ShortURL)

			if err != nil {
				if err.Error() == "sql: no rows in result set" {
					urlData.Err = fmt.Errorf("%w, short url: %s", ErrURLNotFound, url.ShortURL)
				}
			}

			if urlData.Err == nil && createdBy != url.CreatedBy {
				urlData.Err = fmt.Errorf("%w: url author id: %d, request user id: %d", ErrUserIsNotAuthor, createdBy, url.CreatedBy)
			}

			select {
			case <-doneCh:
				return
			case res <- urlData:

			}
		}
	}()
	return res
}

func fanOut(ctx context.Context, doneCh chan struct{}, inCh chan model.URLDelete, repo *DBURLRepository, workersNum int) []chan Result {

	channels := make([]chan Result, workersNum)

	for i := 0; i < workersNum; i++ {
		canBeDeletedCh := canBeDeleted(ctx, doneCh, inCh, repo)
		channels[i] = canBeDeletedCh
	}
	return channels
}

func fanIn(doneCh chan struct{}, resultChs ...chan Result) chan Result {
	finalCh := make(chan Result)

	var wg sync.WaitGroup
	for _, ch := range resultChs {
		wg.Add(1)

		chClosure := ch

		go func() {
			defer wg.Done()

			for data := range chClosure {
				select {
				case <-doneCh:
					return
				case finalCh <- data:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(finalCh)
	}()
	return finalCh
}
