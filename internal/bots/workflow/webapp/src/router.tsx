import {createHashRouter} from "react-router-dom";
import Error from "@/pages/error";
import CardsPage from "@/pages/cards/page.tsx";
import ObjectivePage from "@/pages/objective";
import TaskPage from "@/pages/tasks/page.tsx";
import ScriptFormPage from "@/pages/script-form.tsx";

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
        path: "script",
        element: <ScriptFormPage/>,
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
