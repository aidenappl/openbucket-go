package routers

import (
	"log"
	"net/http"

	"github.com/aidenappl/openbucket-go/handler"
	"github.com/aidenappl/openbucket-go/middleware"
	"github.com/aidenappl/openbucket-go/responder"
	"github.com/aidenappl/openbucket-go/types"
)

func HandleListBuckets(w http.ResponseWriter, r *http.Request) {
	// Get the user session
	session := middleware.RetrieveSession(r)

	// List all buckets
	bucketsList, err := handler.ListBucketsXML(&handler.ListBucketsParams{
		Filter: &handler.ListBucketsParamFilter{
			OwnerID: &session.KeyID,
		},
	})
	if err != nil {
		http.Error(w, "Failed to list buckets", http.StatusInternalServerError)
		log.Println("Error listing buckets:", err)
		return
	}

	// Check if any buckets were found
	if len(bucketsList.Buckets.Bucket) == 0 {
		responder.SendXMLError(w, http.StatusNotFound, "NoSuchBucket", "No buckets found", middleware.GetRequestID(r), middleware.GetHostID(r))
		return
	}

	// Reformat the bucket list
	var responseBuckets types.ListAllMyBucketsResult
	for i := range bucketsList.Buckets.Bucket {
		responseBuckets.Buckets.Bucket = append(responseBuckets.Buckets.Bucket, types.ListAllBucket{
			Name:         bucketsList.Buckets.Bucket[i].Name,
			CreationDate: types.IsoTime(bucketsList.Buckets.Bucket[i].CreationDate),
		})
	}

	responseBuckets.Owner.ID = session.KeyID
	responseBuckets.Owner.DisplayName = session.Name

	responder.SendXML(w, http.StatusOK, responseBuckets)
}
