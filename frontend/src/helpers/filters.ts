const truncate = (text: string, length: number, suffix: string): string => {
  if (text && text.length > length) {
    return text.substring(0, length) + suffix;
  } else {
    return text;
  }
};

const capitalize = (value: string): string => {
  if (!value) {
    return value;
  }

  if (value.length >= 2) {
    return value.charAt(0).toUpperCase() + value.slice(1);
  } else {
    return value;
  }
};

export { truncate, capitalize };
