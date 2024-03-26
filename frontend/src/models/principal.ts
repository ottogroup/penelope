import { DefaultService, UserResponse } from "@/models/api";

export default class Principal {
  private _principal: UserResponse;

  constructor(principal: UserResponse) {
    this._principal = principal;
  }

  async initPrincipal() {
    this._principal = await DefaultService.getUsersMe();
  }

  isValid(): boolean {
    return !!this._principal && !!this._principal.User?.Email && this._principal.RoleBindings?.toString() != "";
  }

  getEmail(): string {
    if (this.isValid()) {
      return this._principal.User?.Email || "";
    }

    return "";
  }
}
