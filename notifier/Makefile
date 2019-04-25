PROJECT := ankimo
ANKIMO_WEBHOOK :=

.PHONY := deploy setenv descenv

deploy:
	gcloud functions deploy Notify --project ${PROJECT} --runtime go111 \
		--trigger-event providers/cloud.firestore/eventTypes/document.create \
		--trigger-resource "projects/ankimo/databases/(default)/documents/StudyRecords/{date}" \
		--set-env-vars="ANKIMO_WEBHOOK"="${ANKIMO_WEBHOOK}"

descenv:
	gcloud functions describe Notify --project ${PROJECT}
