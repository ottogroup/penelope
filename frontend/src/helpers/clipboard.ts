import Notification from "@/models/notification";

const copyToClipboard = (value: string | undefined, addNotification: (_: Notification) => void | null) => {
  if (!value || value.length === 0) {
    return;
  }

  navigator.clipboard.writeText(value).then(
    () => {
      if (addNotification) {
        addNotification(new Notification({ message: "Copied to clipboard!", color: "success" }));
      }
    },
    () => {
      if (addNotification) {
        addNotification(new Notification({ message: "Failed to copy to clipboard!", color: "error" }));
      }
    },
  );
};

export { copyToClipboard };
