import {createHashRouter} from "react-router-dom";
import App from "@/App.tsx";
import Error from "@/pages/Error";

const router = createHashRouter([
  {
    path: "/",
    errorElement: <Error/>,
    children: [
      {
        path: "",
        element: <App/>,
      },
    ]
  },
])

export default router;