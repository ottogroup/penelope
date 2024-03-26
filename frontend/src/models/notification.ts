type notification = {
  id?: string;
  model?: boolean;
  message: string;
  isError?: boolean;
  color?: string;
  visitHome?: boolean;
  historyBack?: boolean;
  timeout?: number;
  position?: number;
};

export default class Notification {
  private _id: string;
  private _model: boolean;
  private _message: string;
  private _isError: boolean;
  private _color: string;
  private _visitHome: boolean;
  private _historyBack: boolean;
  private _timeout: number;
  private _position: number;

  constructor(options: notification) {
    this._id = options.id ?? "";
    this._model = options.model ?? false;
    this._message = options.message ?? "";
    this._isError = options.isError ?? false;
    this._color = options.color ?? "success";
    this._visitHome = options.visitHome ?? false;
    this._historyBack = options.historyBack ?? false;
    this._timeout = options.timeout ?? 5000;
    this._position = options.position ?? 0;
  }

  get id(): string {
    return this._id;
  }

  set id(value: string) {
    this._id = value;
  }

  get model(): boolean {
    return this._model;
  }

  set model(value: boolean) {
    this._model = value;
  }

  get message(): string {
    return this._message;
  }

  set message(value: string) {
    this._message = value;
  }

  get isError(): boolean {
    return this._isError;
  }

  set isError(value: boolean) {
    this._isError = value;
  }

  get color(): string {
    return this._color;
  }

  set color(value: string) {
    this._color = value;
  }

  get visitHome(): boolean {
    return this._visitHome;
  }

  set visitHome(value: boolean) {
    this._visitHome = value;
  }

  get historyBack(): boolean {
    return this._historyBack;
  }

  set historyBack(value: boolean) {
    this._historyBack = value;
  }

  get timeout(): number {
    return this._timeout;
  }

  set timeout(value: number) {
    this._timeout = value;
  }

  get position(): number {
    return this._position;
  }

  set position(value: number) {
    this._position = value;
  }
}
