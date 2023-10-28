import {createHashRouter} from "react-router-dom";
import Error from "@/pages/Error";
import CardsPage from "@/pages/cards/page.tsx";
import ObjectivesPage from "@/pages/Objectives";
import ObjectivePage from "@/pages/objective";
import ObjectiveFormPage from "@/pages/objective-form";

const router = createHashRouter([
  {
    path: "/",
    errorElement: <Error/>,
    children: [
      {
        path: "",
        element: <ObjectivesPage/>,
      },
      {
        path: "obj",
        element: <ObjectiveFormPage/>,
      },
      {
        path: "obj/:sequence",
        element: <ObjectivePage/>,
      },
      {
        path: "demo",
        element: <CardsPage/>,
      },
    ]
  },
])

export default router;
