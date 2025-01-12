from flask import Blueprint, request
from flask import current_app as app
from utils.response import set_response
from utils.custom_exception import InvalidUsage
from datetime import datetime
import os
from utils.helper import rmdir_recursive
from models.notification import RunningNotification

internal_api = Blueprint("internal_api", __name__)


@internal_api.route('/ping', methods=["GET"])
def ping():
    return set_response({"message": "pong"})


@internal_api.route("/running_notification", methods=["POST"])
def add_running_notification():
    if not request.is_json:
        raise InvalidUsage("Missing json in request")
    if type(request.json) != dict:
        raise InvalidUsage("Request data invalid")
    content = request.json.get('content', '')
    source_application_id = request.json.get('source_application_id', '')
    if len(source_application_id) == 0:
        raise InvalidUsage("source_application_id is mandatory")
    expiry_in_secs = request.json.get('expiry_in_secs', None)
    try:
        r_notification = RunningNotification.query.filter_by(source_application_id=source_application_id).one_or_none()
        if r_notification is None:
            r_notification = RunningNotification(
                content=content,
                source_application_id=source_application_id,
                expiry_in_secs=expiry_in_secs
            )
        else:
            r_notification.content = content
            r_notification.expiry_in_secs = expiry_in_secs
            r_notification.updated_at = datetime.now()
        r_notification.save()
    except Exception as ex:
        print(ex)
    return set_response("OK")


@internal_api.route("/clean_agent_logs", methods=["POST"], endpoint="api_v1_5_delete_agent_logs")
def delete_agent_logs():
    """
    Clean up agent diagnostic logs
    """
    payloads = request.json
    path = str(payloads.get("path", ""))
    if path.startswith("/tmp/deepfence-logs"):
        if os.path.isdir(path):
            rmdir_recursive(path)
    return set_response(data="Ok")
