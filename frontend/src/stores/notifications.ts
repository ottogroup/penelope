import { defineStore } from "pinia";
import Notification from "@/models/notification";
import { ref, Ref } from "vue";
import { ApiError } from "@/models/api";

type genericError = { error: { code: number, message: string} };
const isGenericError = (err: unknown): err is genericError =>
  <genericError>err &&
  !!err &&
  typeof err === "object" &&
  "error" in err &&
  Object.prototype.hasOwnProperty.call(err, "error") &&
  typeof err.error === "object" &&
  Object.prototype.hasOwnProperty.call(err.error, "code") &&
  Object.prototype.hasOwnProperty.call(err.error, "message");

export const useNotificationsStore = defineStore("notifications", () => {
  const notifications = ref<Notification[]>([]);
  const errorMessage: Ref<string> = ref("");
  const notificationSnackbarSize = 88 + 5;

  function handleError(err: string | genericError | ApiError) {
    console.error(typeof err, err);

    let message;
    if (err instanceof ApiError) {
      message = `Api call finished with status code ${err.status} and message: ${err.message}`;
    } else if (isGenericError(err)) {
      message = `Api call finished with status code ${err.error.code} and message: ${err.error.message}`;
    } else {
      message = "Error during api call: " + err;
    }

    addNotification(
      new Notification({
        message: message,
        color: "error",
      }),
    );
  }

  function showURLLengthExceededNotification(maxLength: number) {
    addNotification(
      new Notification({
        message: `Current selection exceed the maximum URL length of ${maxLength} characters and
          selection could not be applied.`,
        color: "error",
      }),
    );
  }

  function addNotification(n: Notification) {
    n.position = notificationSnackbarSize * notifications.value.length;
    n.id = crypto.randomUUID();
    n.model = true;
    notifications.value.push(n);
  }

  function removeNotification(id: string) {
    const removedIdx = notifications.value.findIndex((x) => x.id === id);
    notifications.value.splice(removedIdx, 1);
    notifications.value.forEach((x, idx) => (x.position = notificationSnackbarSize * idx));
  }

  function setErrorMessage(m: string) {
    errorMessage.value = m;
  }

  function clearErrorMessage() {
    errorMessage.value = "";
  }

  return {
    notifications,
    errorMessage,
    addNotification,
    handleError,
    removeNotification,
    setErrorMessage,
    clearErrorMessage,
    showURLLengthExceededNotification,
  };
});
