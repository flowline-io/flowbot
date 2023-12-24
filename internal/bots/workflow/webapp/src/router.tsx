import {createHashRouter} from "react-router-dom";
import Error from "@/pages/error";
import CardsPage from "@/pages/cards/page.tsx";
import ObjectivePage from "@/pages/objective";
import ObjectiveFormPage from "@/pages/objective-form";
import TaskPage from "@/pages/tasks/page.tsx";

const router = createHashRouter([
  {
    path: "/",
    errorElement: <Error/>,
    children: [
      {
        path: "",
        element: <TaskPage/>,
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
