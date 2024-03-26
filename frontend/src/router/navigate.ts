import { NavigationFailure, NavigationFailureType, RouteLocationRaw, Router, isNavigationFailure } from "vue-router";

// by convention, composable function names start with "use"
export function useNavigate() {
  function navigateTo(router: Router, location: RouteLocationRaw): Promise<NavigationFailure | void | undefined> {
    const prom = router.push(location);
    prom.catch((err) => {
      if (!isNavigationFailure(err, NavigationFailureType.duplicated)) {
        console.error(err);
      }
    });
    return prom;
  }

  return navigateTo;
}
