import {createHashRouter} from "react-router-dom";
import Error from "@/pages/error";
import CardsPage from "@/pages/cards/page.tsx";
import TaskPage from "@/pages/tasks/page.tsx";
import ScriptFormPage from "@/pages/script-form.tsx";
import JobPage from "@/pages/jobs/page.tsx";

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
        path: "workflow/:id",
        element: <ScriptFormPage/>,
      },
      {
        path: "workflow/:id/jobs",
        element: <JobPage/>,
      }
    ]
  },
])

export default router;
