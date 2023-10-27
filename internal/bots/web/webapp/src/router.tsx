import {createHashRouter} from "react-router-dom";
import App from "@/App.tsx";
import Error from "@/pages/Error";
import CardsPage from "@/pages/cards/page.tsx";

const router = createHashRouter([
  {
    path: "/",
    errorElement: <Error/>,
    children: [
      {
        path: "",
        element: <App/>,
      },
      {
        path: "demo",
        element: <CardsPage/>,
      },
    ]
  },
])

export default router;