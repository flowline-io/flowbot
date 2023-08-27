import {createHashRouter} from "react-router-dom";
import App from "./page/App";
import Error from "./page/Error";
import Form from "./page/Form";
import Objective from "./page/Objective";
import ObjectiveForm from "./page/ObjectiveForm";

const router = createHashRouter([
  {
    path: "/",
    errorElement: <Error />,
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