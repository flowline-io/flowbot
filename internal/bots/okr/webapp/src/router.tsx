import {createHashRouter} from "react-router-dom";
import App from "./pages/App";
import Error from "./pages/Error";
import Form from "./pages/Form";
import Objective from "./pages/Objective";
import ObjectiveForm from "./pages/ObjectiveForm";

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
        path: "form",
        element: <Form/>,
      },
      {
        path: "obj/:id",
        element: <Objective/>,
      },
      {
        path: "obj",
        element: <ObjectiveForm/>,
      },
    ]
  },
]);

export default router;