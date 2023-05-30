import NavigationBar from "./NavigationBar.tsx";

export default function Header(props: { title: string; active: string }) {
  return (
    <div>
      <header class="mx-auto max-w-screen-lg flex gap-3 justify-end">
        <NavigationBar class="hidden md:flex" active={props.active} />
      </header>
    </div>
  );
}
